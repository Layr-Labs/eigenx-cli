package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	kmscrypto "github.com/Layr-Labs/eigenx-kms/pkg/crypto"
	kmstypes "github.com/Layr-Labs/eigenx-kms/pkg/types"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

const MAX_ADDRESS_COUNT = 5

type AppStatusResponse struct {
	Apps []AppStatus `json:"apps"`
}

type AppStatus struct {
	Status string `json:"app_status"`
}

type RawAppInfoResponse struct {
	Apps []RawAppInfo `json:"apps"`
}

// InstanceType represents a machine instance type configuration.
// This struct matches the backend API response format for SKUs.
type InstanceType struct {
	SKU         string `json:"sku"`         // SKU value (e.g., "g1-standard-4t")
	Description string `json:"description"` // Human-readable description (e.g., "4 vCPUs, 16 GB memory, TDX")
}

type SKUListResponse struct {
	SKUs []InstanceType `json:"skus"`
}

type RawAppInfo struct {
	Addresses   json.RawMessage `json:"addresses"`
	Status      string          `json:"app_status"`
	Ip          string          `json:"ip"`
	MachineType string          `json:"machine_type"`
}

// AppInfo contains the app info with parsed and validated addresses
type AppInfo struct {
	EVMAddresses    []kmstypes.EVMAddressAndDerivationPath
	SolanaAddresses []kmstypes.SolanaAddressAndDerivationPath
	Status          string
	Ip              string
	MachineType     string
}

type AppInfoResponse struct {
	Apps []AppInfo
}

type UserApiClient struct {
	environmentConfig common.EnvironmentConfig
	Client            *http.Client
}

func NewUserApiClient(cCtx *cli.Context) (*UserApiClient, error) {
	environmentConfig, err := GetEnvironmentConfig(cCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment config: %w", err)
	}

	return &UserApiClient{
		environmentConfig: environmentConfig,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (cc *UserApiClient) GetStatuses(cCtx *cli.Context, appIDs []ethcommon.Address) (*AppStatusResponse, error) {
	endpoint := fmt.Sprintf("%s/status", cc.environmentConfig.UserApiServerURL)

	// Build query parameters
	params := url.Values{}
	params.Add("apps", buildAppIDsParam(appIDs))

	fullURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	resp, err := cc.makeAuthenticatedRequest(cCtx, "GET", fullURL, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, handleErrorResponse(resp)
	}

	var result AppStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (cc *UserApiClient) GetInfos(cCtx *cli.Context, appIDs []ethcommon.Address, addressCount int) (*AppInfoResponse, error) {
	if addressCount > MAX_ADDRESS_COUNT {
		addressCount = MAX_ADDRESS_COUNT
	}

	endpoint := fmt.Sprintf("%s/info", cc.environmentConfig.UserApiServerURL)

	// Build query parameters
	params := url.Values{}
	params.Add("apps", buildAppIDsParam(appIDs))

	fullURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	resp, err := cc.makeAuthenticatedRequest(cCtx, "GET", fullURL, &common.CanViewSensitiveAppInfoPermission)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, handleErrorResponse(resp)
	}

	var rawResult RawAppInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&rawResult); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Get signing key for verification
	_, signingKey, err := getKMSKeysForEnvironment(cc.environmentConfig.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get KMS keys: %w", err)
	}

	// Process each app's addresses
	appIDStrings := buildAppIDsParam(appIDs)
	appIDList := strings.Split(appIDStrings, ",")

	result := &AppInfoResponse{
		Apps: make([]AppInfo, len(rawResult.Apps)),
	}

	for i, rawApp := range rawResult.Apps {
		evmAddrs, solanaAddrs, err := processAddressesResponse(
			rawApp.Addresses,
			appIDs[i],
			signingKey,
			addressCount,
		)
		if err != nil {
			return nil, fmt.Errorf("error processing addresses for app %s: %w", appIDList[i], err)
		}

		result.Apps[i] = AppInfo{
			EVMAddresses:    evmAddrs,
			SolanaAddresses: solanaAddrs,
			Status:          rawApp.Status,
			Ip:              rawApp.Ip,
			MachineType:     rawApp.MachineType,
		}
	}

	return result, nil
}

func (cc *UserApiClient) GetLogs(cCtx *cli.Context, appID ethcommon.Address) (string, error) {
	endpoint := fmt.Sprintf("%s/logs/%s", cc.environmentConfig.UserApiServerURL, appID.Hex())

	resp, err := cc.makeAuthenticatedRequest(cCtx, "GET", endpoint, &common.CanViewAppLogsPermission)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", handleErrorResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

func (cc *UserApiClient) GetSKUs(cCtx *cli.Context) (*SKUListResponse, error) {
	endpoint := fmt.Sprintf("%s/skus", cc.environmentConfig.UserApiServerURL)

	resp, err := cc.makeAuthenticatedRequest(cCtx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, handleErrorResponse(resp)
	}

	var result SKUListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode SKU list response: %w", err)
	}

	return &result, nil
}

// buildAppIDsParam creates a comma-separated string of app IDs for URL parameters
func buildAppIDsParam(appIDs []ethcommon.Address) string {
	appIDStrings := make([]string, len(appIDs))
	for i, appID := range appIDs {
		appIDStrings[i] = appID.Hex()
	}
	return strings.Join(appIDStrings, ",")
}

// makeAuthenticatedRequest performs an HTTP request with optional authentication
func (cc *UserApiClient) makeAuthenticatedRequest(cCtx *cli.Context, method, url string, permission *[4]byte) (*http.Response, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add auth headers if permission is specified
	if permission != nil {
		expiry := big.NewInt(time.Now().Add(5 * time.Minute).Unix())
		authHeaders, err := GenerateAuthHeaders(cCtx, *permission, expiry)
		if err != nil {
			return nil, fmt.Errorf("failed to generate auth headers: %w", err)
		}

		for key, value := range authHeaders {
			req.Header.Set(key, value)
		}
	}

	resp, err := cc.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to %s: %w", url, err)
	}

	return resp, nil
}

// handleErrorResponse processes non-200 HTTP responses with standard error parsing
func handleErrorResponse(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	// Try to parse JSON error response
	var errorResp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error != "" {
		return fmt.Errorf("userApi server error: %s", errorResp.Error)
	}

	// Fallback to raw body if not valid JSON
	return fmt.Errorf("userApi server returned status %d: %s", resp.StatusCode, string(body))
}

// processAddressesResponse attempts to parse and validate addresses response as V2, then V1
func processAddressesResponse(
	rawAddresses json.RawMessage,
	appID ethcommon.Address,
	signingKey []byte,
	addressCount int,
) (evmAddrs []kmstypes.EVMAddressAndDerivationPath, solanaAddrs []kmstypes.SolanaAddressAndDerivationPath, err error) {
	// Try V2 first - unmarshal and verify signature
	var signedV2 kmstypes.SignedResponse[kmstypes.AddressesResponseV2]
	if err := json.Unmarshal(rawAddresses, &signedV2); err == nil {
		// Verify signature - if this succeeds, it's V2
		if ok, err := kmscrypto.VerifyKMSSignature(signedV2, signingKey); err == nil && ok {
			// Validate AppID matches
			if ethcommon.HexToAddress(signedV2.Data.AppID).Cmp(appID) != 0 {
				return nil, nil, fmt.Errorf("app ID mismatch in V2 response")
			}

			// Truncate to requested count
			evmAddrs = signedV2.Data.EVMAddresses
			if len(evmAddrs) > addressCount {
				evmAddrs = evmAddrs[:addressCount]
			}
			solanaAddrs = signedV2.Data.SolanaAddresses
			if len(solanaAddrs) > addressCount {
				solanaAddrs = solanaAddrs[:addressCount]
			}
			return evmAddrs, solanaAddrs, nil
		}
		// Signature failed - might be V1 response, fall through to try V1
	}

	// Try V1 fallback
	var signedV1 kmstypes.SignedResponse[kmstypes.AddressesResponseV1]
	if err := json.Unmarshal(rawAddresses, &signedV1); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal as V1 or V2: %w", err)
	}

	// Verify signature
	if ok, err := kmscrypto.VerifyKMSSignature(signedV1, signingKey); err != nil {
		return nil, nil, fmt.Errorf("error verifying V1 signature: %w", err)
	} else if !ok {
		return nil, nil, fmt.Errorf("invalid V1 signature")
	}

	// V1 doesn't have AppID field, so we can't validate it
	// Truncate to requested count
	evmAddrs = signedV1.Data.EVMAddresses
	if len(evmAddrs) > addressCount {
		evmAddrs = evmAddrs[:addressCount]
	}
	solanaAddrs = signedV1.Data.SolanaAddresses
	if len(solanaAddrs) > addressCount {
		solanaAddrs = solanaAddrs[:addressCount]
	}
	return evmAddrs, solanaAddrs, nil
}
