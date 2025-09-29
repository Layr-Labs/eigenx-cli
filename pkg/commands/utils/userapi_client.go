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

type AppInfoResponse struct {
	Apps []AppInfo `json:"apps"`
}

type AppInfo struct {
	Addresses kmstypes.SignedResponse[kmstypes.AddressesResponse] `json:"addresses"`
	Status    string                                              `json:"app_status"`
	Ip        string                                              `json:"ip"`
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

	var result AppInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// verify signature
	_, signingKey, err := getKMSKeysForEnvironment(cc.environmentConfig.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get KMS keys: %w", err)
	}

	appIDStrings := buildAppIDsParam(appIDs)
	appIDList := strings.Split(appIDStrings, ",")
	for i, appInfo := range result.Apps {
		ok, err := kmscrypto.VerifyKMSSignature(appInfo.Addresses, signingKey)
		if err != nil {
			return nil, fmt.Errorf("errors verifying signature for app %s: %w", appIDList[i], err)
		}
		if !ok {
			return nil, fmt.Errorf("invalid signature for app %s", appIDList[i])
		}

		result.Apps[i].Addresses.Data.EVMAddresses = result.Apps[i].Addresses.Data.EVMAddresses[:addressCount]
		result.Apps[i].Addresses.Data.SolanaAddresses = result.Apps[i].Addresses.Data.SolanaAddresses[:addressCount]
	}

	return &result, nil
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
