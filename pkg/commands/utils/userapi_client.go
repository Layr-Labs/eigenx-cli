package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Layr-Labs/eigenx-cli/internal/version"
	"github.com/Layr-Labs/eigenx-cli/pkg/common"
	kmscrypto "github.com/Layr-Labs/eigenx-kms/pkg/crypto"
	kmstypes "github.com/Layr-Labs/eigenx-kms/pkg/types"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

// SubscriptionStatus represents the state of a user's subscription
type SubscriptionStatus string

const (
	StatusIncomplete        SubscriptionStatus = "incomplete"
	StatusIncompleteExpired SubscriptionStatus = "incomplete_expired"
	StatusTrialing          SubscriptionStatus = "trialing"
	StatusActive            SubscriptionStatus = "active"
	StatusPastDue           SubscriptionStatus = "past_due"
	StatusCanceled          SubscriptionStatus = "canceled"
	StatusUnpaid            SubscriptionStatus = "unpaid"
	StatusPaused            SubscriptionStatus = "paused"
	StatusInactive          SubscriptionStatus = "inactive"
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

type CheckoutSessionResponse struct {
	CheckoutURL string `json:"checkout_url"`
}

type UpcomingInvoice struct {
	Amount      float64 `json:"amount"`
	Date        int64   `json:"date"`
	Description string  `json:"description"`
}

type UserSubscriptionResponse struct {
	Status             SubscriptionStatus `json:"status"`
	CurrentPeriodStart *int64             `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   *int64             `json:"current_period_end,omitempty"`
	PlanPrice          *float64           `json:"plan_price,omitempty"`
	Currency           *string            `json:"currency,omitempty"`
	UpcomingInvoice    *UpcomingInvoice   `json:"upcoming_invoice,omitempty"`
	CancelAtPeriodEnd  *bool              `json:"cancel_at_period_end,omitempty"`
	CanceledAt         *int64             `json:"canceled_at,omitempty"`
	PortalURL          *string            `json:"portal_url,omitempty"`
}

type AppProfileResponse struct {
	Name        string  `json:"name"`
	Website     *string `json:"website,omitempty"`
	Description *string `json:"description,omitempty"`
	XURL        *string `json:"xURL,omitempty"`
	ImageURL    *string `json:"imageURL,omitempty"`
}

type RawAppInfo struct {
	Addresses   json.RawMessage     `json:"addresses"`
	Status      string              `json:"app_status"`
	Ip          string              `json:"ip"`
	MachineType string              `json:"machine_type"`
	Profile     *AppProfileResponse `json:"profile,omitempty"`
}

// AppInfo contains the app info with parsed and validated addresses
type AppInfo struct {
	EVMAddresses    []kmstypes.EVMAddressAndDerivationPath
	SolanaAddresses []kmstypes.SolanaAddressAndDerivationPath
	Status          string
	Ip              string
	MachineType     string
	Profile         *AppProfileResponse
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

	resp, err := cc.makeAuthenticatedRequest(cCtx, "GET", fullURL, nil, "", nil)
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

	resp, err := cc.makeAuthenticatedRequest(cCtx, "GET", fullURL, nil, "", &common.CanViewSensitiveAppInfoPermission)
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
			Profile:         rawApp.Profile,
		}
	}

	return result, nil
}

func (cc *UserApiClient) GetLogs(cCtx *cli.Context, appID ethcommon.Address) (string, error) {
	endpoint := fmt.Sprintf("%s/logs/%s", cc.environmentConfig.UserApiServerURL, appID.Hex())

	resp, err := cc.makeAuthenticatedRequest(cCtx, "GET", endpoint, nil, "", &common.CanViewAppLogsPermission)
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

	resp, err := cc.makeAuthenticatedRequest(cCtx, "GET", endpoint, nil, "", nil)
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

func (cc *UserApiClient) CreateCheckoutSession(cCtx *cli.Context) (*CheckoutSessionResponse, error) {
	endpoint := fmt.Sprintf("%s/subscription", cc.environmentConfig.UserApiServerURL)

	resp, err := cc.makeAuthenticatedRequest(cCtx, "POST", endpoint, nil, "", &common.CanManageBillingPermission)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, handleErrorResponse(resp)
	}

	var result CheckoutSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (cc *UserApiClient) GetUserSubscription(cCtx *cli.Context) (*UserSubscriptionResponse, error) {
	endpoint := fmt.Sprintf("%s/subscription", cc.environmentConfig.UserApiServerURL)

	resp, err := cc.makeAuthenticatedRequest(cCtx, "GET", endpoint, nil, "", &common.CanManageBillingPermission)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, handleErrorResponse(resp)
	}

	var result UserSubscriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (cc *UserApiClient) CancelSubscription(cCtx *cli.Context) error {
	endpoint := fmt.Sprintf("%s/subscription", cc.environmentConfig.UserApiServerURL)

	resp, err := cc.makeAuthenticatedRequest(cCtx, "DELETE", endpoint, nil, "", &common.CanManageBillingPermission)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return handleErrorResponse(resp)
	}

	return nil
}

// UploadAppProfile uploads app profile metadata with optional image
func (cc *UserApiClient) UploadAppProfile(cCtx *cli.Context, appAddress string, name string, website, description, xURL *string, imagePath string) (*AppProfileResponse, error) {
	endpoint := fmt.Sprintf("%s/apps/%s/profile", cc.environmentConfig.UserApiServerURL, appAddress)

	// Create multipart form body
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add required name field
	if err := writer.WriteField("name", name); err != nil {
		return nil, fmt.Errorf("failed to write name field: %w", err)
	}

	// Add optional text fields
	if website != nil && *website != "" {
		if err := writer.WriteField("website", *website); err != nil {
			return nil, fmt.Errorf("failed to write website field: %w", err)
		}
	}

	if description != nil && *description != "" {
		if err := writer.WriteField("description", *description); err != nil {
			return nil, fmt.Errorf("failed to write description field: %w", err)
		}
	}

	if xURL != nil && *xURL != "" {
		if err := writer.WriteField("xURL", *xURL); err != nil {
			return nil, fmt.Errorf("failed to write xURL field: %w", err)
		}
	}

	// Add optional image file
	if imagePath != "" {
		file, err := os.Open(imagePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open image file: %w", err)
		}
		defer file.Close()

		part, err := writer.CreateFormFile("image", filepath.Base(imagePath))
		if err != nil {
			return nil, fmt.Errorf("failed to create form file: %w", err)
		}

		if _, err := io.Copy(part, file); err != nil {
			return nil, fmt.Errorf("failed to copy image data: %w", err)
		}
	}

	// Close the multipart writer to finalize the form
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Use makeAuthenticatedRequest to handle authentication
	resp, err := cc.makeAuthenticatedRequest(cCtx, "POST", endpoint, body, writer.FormDataContentType(), &common.CanUpdateAppProfilePermission)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check for success (201 Created or 200 OK)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, handleErrorResponse(resp)
	}

	// Parse response
	var result AppProfileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
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

// makeAuthenticatedRequest performs an HTTP request with optional authentication and body
// contentType parameter allows setting custom Content-Type header (e.g., for multipart forms)
func (cc *UserApiClient) makeAuthenticatedRequest(cCtx *cli.Context, method, url string, body io.Reader, contentType string, permission *[4]byte) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add x-client-id header to identify the CLI client
	clientID := fmt.Sprintf("eigenx-cli/%s", version.GetVersion())
	req.Header.Set("x-client-id", clientID)

	// Set content type if provided
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
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
