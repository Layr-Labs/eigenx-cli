package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	// GCE metadata server for instance identity tokens
	metadataServerURL = "http://metadata.google.internal/computeMetadata/v1"
	identityEndpoint  = "/instance/service-accounts/default/identity"
	instanceHeader    = "X-Instance-Token"
)

// RemoteCertificateStorage represents the certificate data sent to the API
type RemoteCertificateStorage struct {
	Certificate string `json:"certificate"`
}

// CertificateMetadata represents the metadata structure from the API
type CertificateMetadata struct {
	IssuedAt    time.Time `json:"issued_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	Fingerprint string    `json:"fingerprint"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RemoteCertificateResponse represents the API response for GET operations
type RemoteCertificateResponse struct {
	Certificate string              `json:"certificate"`
	Metadata    CertificateMetadata `json:"metadata"`
}

// RemoteStorage implements Storage interface for remote API persistence
type RemoteStorage struct {
	BaseURL       string
	TokenAudience string // Audience for GCE identity token
	Client        *http.Client
	Log           *slog.Logger
}

// NewRemoteStorage creates a new remote storage client
func NewRemoteStorage(baseURL string, audience string, log *slog.Logger) *RemoteStorage {
	return &RemoteStorage{
		BaseURL:       baseURL,
		TokenAudience: audience,
		Log:           log,
	}
}

// Load retrieves a certificate from remote storage
//
// Returns nil chain if certificate doesn't exist.
func (r *RemoteStorage) Load(domain string) (ChainPEM, Metadata, error) {
	if r.BaseURL == "" {
		return nil, Metadata{}, nil
	}

	token, err := r.fetchGCEToken()
	if err != nil {
		return nil, Metadata{}, fmt.Errorf("get GCE identity token: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/certs/%s", r.BaseURL, domain)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, Metadata{}, err
	}
	req.Header.Set(instanceHeader, token)

	resp, err := r.httpClient().Do(req)
	if err != nil {
		return nil, Metadata{}, fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, Metadata{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, Metadata{}, fmt.Errorf("remote GET status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var apiResp RemoteCertificateResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, Metadata{}, fmt.Errorf("decode response: %w", err)
	}

	// Check if certificate is present
	if len(apiResp.Certificate) == 0 {
		return nil, Metadata{}, nil
	}

	r.Log.Info("loaded certificate from remote storage", "expires", apiResp.Metadata.ExpiresAt.Format(time.RFC3339))

	return ChainPEM(apiResp.Certificate), Metadata{
		IssuedAt:  apiResp.Metadata.IssuedAt,
		ExpiresAt: apiResp.Metadata.ExpiresAt,
	}, nil
}

// Store persists a certificate to remote storage
//
// No-op if BaseURL is empty.
//
// TODO: Future consideration for multi-instance deployments:
// Currently, this implementation assumes a single instance is issuing certificates
// at any given time. If multiple instances need to issue certificates concurrently,
// a distributed locking mechanism should be implemented to prevent duplicate
// certificate requests and potential rate limiting issues. This could be achieved
// through compare-and-swap operations or a distributed lock service.
// For now, single-instance operation is sufficient for our use case.
func (r *RemoteStorage) Store(domain string, chain ChainPEM) error {
	if r.BaseURL == "" {
		return nil
	}

	token, err := r.fetchGCEToken()
	if err != nil {
		return fmt.Errorf("get GCE identity token: %w", err)
	}

	payload, err := json.Marshal(RemoteCertificateStorage{
		Certificate: string(chain),
	})
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/certs/%s", r.BaseURL, domain)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set(instanceHeader, token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remote POST status=%d body=%s", resp.StatusCode, string(body))
	}

	r.Log.Info("stored certificate to remote storage")
	return nil
}

// fetchGCEToken retrieves a GCE instance identity token from the metadata server
func (r *RemoteStorage) fetchGCEToken() (string, error) {
	// Build the request URL with query parameters
	url := fmt.Sprintf("%s%s?audience=%s&format=full",
		metadataServerURL, identityEndpoint, r.TokenAudience)

	// Create request with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("creating metadata request: %w", err)
	}

	// Add required metadata header
	req.Header.Set("Metadata-Flavor", "Google")

	// Execute request
	resp, err := r.httpClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("metadata server request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("metadata server returned %d: %s",
			resp.StatusCode, strings.TrimSpace(string(body)))
	}

	// Read token from response body
	tokenBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading token response: %w", err)
	}

	token := strings.TrimSpace(string(tokenBytes))
	if token == "" {
		return "", fmt.Errorf("metadata server returned empty token")
	}

	return token, nil
}

// httpClient returns the HTTP client to use, creating one if necessary
func (r *RemoteStorage) httpClient() *http.Client {
	if r.Client == nil {
		r.Client = &http.Client{
			Timeout: 10 * time.Second,
		}
	}
	return r.Client
}
