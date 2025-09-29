package storage

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// mockHTTPClient records requests and returns configured responses
type mockHTTPTransport struct {
	requests  []*http.Request
	responses []mockResponse
	callCount int
}

type mockResponse struct {
	status int
	body   string
	err    error
}

func (m *mockHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone request to avoid mutation issues
	clonedReq := req.Clone(req.Context())
	if req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		req.Body.Close()
		clonedReq.Body = io.NopCloser(bytes.NewReader(body))
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	m.requests = append(m.requests, clonedReq)

	if m.callCount >= len(m.responses) {
		return nil, http.ErrNotSupported
	}
	resp := m.responses[m.callCount]
	m.callCount++
	if resp.err != nil {
		return nil, resp.err
	}
	return &http.Response{
		StatusCode: resp.status,
		Body:       io.NopCloser(strings.NewReader(resp.body)),
		Request:    req,
	}, nil
}

// TestRemoteStorageLoad tests loading certificates from remote storage
func TestRemoteStorageLoad(t *testing.T) {
	tests := []struct {
		name      string
		domain    string
		responses []mockResponse
		wantChain string
		wantMeta  Metadata
		wantErr   bool
	}{
		{
			name:   "successful load",
			domain: "example.com",
			responses: []mockResponse{
				{
					status: 200,
					body: `{
						"certificate": "-----BEGIN CERTIFICATE-----\ntest cert\n-----END CERTIFICATE-----",
						"metadata": {
							"expires_at": "2024-12-31T23:59:59Z",
							"issued_at": "2024-01-01T00:00:00Z",
							"fingerprint": "test-fingerprint",
							"created_at": "2024-01-01T00:00:00Z",
							"updated_at": "2024-01-01T00:00:00Z"
						}
					}`,
				},
			},
			wantChain: "-----BEGIN CERTIFICATE-----\ntest cert\n-----END CERTIFICATE-----",
			wantMeta: Metadata{
				ExpiresAt: time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			},
		},
		{
			name:   "not found",
			domain: "notfound.com",
			responses: []mockResponse{
				{status: 404, body: `{"error": "not found"}`},
			},
			wantChain: "", // Returns nil chain for not found
			wantErr:   false,
		},
		{
			name:   "server error",
			domain: "error.com",
			responses: []mockResponse{
				{status: 500, body: "Internal Server Error"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock both the GCE metadata server and the API server
			transport := &mockHTTPTransport{
				responses: append([]mockResponse{
					// First response is always for the GCE token fetch
					{status: 200, body: "test-gce-jwt-token"},
				}, tt.responses...),
			}
			storage := &RemoteStorage{
				BaseURL:       "https://api.test.com",
				TokenAudience: "test-audience",
				Client:        &http.Client{Transport: transport},
				Log:           slog.Default(),
			}

			chain, meta, err := storage.Load(tt.domain)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				if string(chain) != tt.wantChain {
					t.Errorf("Load() chain = %q, want %q", chain, tt.wantChain)
				}
				if !meta.ExpiresAt.Equal(tt.wantMeta.ExpiresAt) {
					t.Errorf("Load() meta.ExpiresAt = %v, want %v", meta.ExpiresAt, tt.wantMeta.ExpiresAt)
				}
			}

			// Verify requests
			if len(transport.requests) != 2 {
				t.Fatalf("expected 2 requests, got %d", len(transport.requests))
			}

			// Verify GCE token request
			gceReq := transport.requests[0]
			if !strings.Contains(gceReq.URL.String(), "metadata.google.internal") {
				t.Errorf("first request should be to metadata server: %s", gceReq.URL)
			}

			// Verify API request
			apiReq := transport.requests[1]
			if apiReq.Method != "GET" {
				t.Errorf("API request method = %s, want GET", apiReq.Method)
			}
			if !strings.Contains(apiReq.URL.Path, tt.domain) {
				t.Errorf("API request path %s doesn't contain domain %s", apiReq.URL.Path, tt.domain)
			}
			if apiReq.Header.Get("X-Instance-Token") != "test-gce-jwt-token" {
				t.Errorf("missing or incorrect instance token header")
			}
		})
	}
}

// TestRemoteStorageStore tests storing certificates to remote storage
func TestRemoteStorageStore(t *testing.T) {
	testChain := ChainPEM("-----BEGIN CERTIFICATE-----\ntest cert\n-----END CERTIFICATE-----")

	tests := []struct {
		name      string
		domain    string
		responses []mockResponse
		wantErr   bool
	}{
		{
			name:   "successful store",
			domain: "example.com",
			responses: []mockResponse{
				{status: 200, body: `{"success": true}`},
			},
			wantErr: false,
		},
		{
			name:   "unauthorized",
			domain: "example.com",
			responses: []mockResponse{
				{status: 401, body: `{"error": "unauthorized"}`},
			},
			wantErr: true,
		},
		{
			name:   "server error",
			domain: "example.com",
			responses: []mockResponse{
				{status: 500, body: "Internal Server Error"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock both the GCE metadata server and the API server
			transport := &mockHTTPTransport{
				responses: append([]mockResponse{
					// First response is always for the GCE token fetch
					{status: 200, body: "test-gce-jwt-token"},
				}, tt.responses...),
			}
			storage := &RemoteStorage{
				BaseURL:       "https://api.test.com",
				TokenAudience: "test-audience",
				Client:        &http.Client{Transport: transport},
				Log:           slog.Default(),
			}

			err := storage.Store(tt.domain, testChain)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Store() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify requests
			if len(transport.requests) != 2 {
				t.Fatalf("expected 2 requests, got %d", len(transport.requests))
			}

			// Verify GCE token request
			gceReq := transport.requests[0]
			if !strings.Contains(gceReq.URL.String(), "metadata.google.internal") {
				t.Errorf("first request should be to metadata server: %s", gceReq.URL)
			}

			// Verify API request
			apiReq := transport.requests[1]
			if apiReq.Method != "POST" {
				t.Errorf("API request method = %s, want POST", apiReq.Method)
			}
			if apiReq.Header.Get("X-Instance-Token") != "test-gce-jwt-token" {
				t.Errorf("missing or incorrect instance token header")
			}

			// Verify URL contains domain
			if !strings.Contains(apiReq.URL.Path, tt.domain) {
				t.Errorf("API request path %s doesn't contain domain %s", apiReq.URL.Path, tt.domain)
			}

			// Verify request body
			body, _ := io.ReadAll(apiReq.Body)
			var payload map[string]interface{}
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("failed to parse request body: %v", err)
			}
			if payload["certificate"] != string(testChain) {
				t.Errorf("request certificate mismatch")
			}
		})
	}
}

// TestRemoteStorageLoadBodyRead tests reading response body
func TestRemoteStorageLoadBodyRead(t *testing.T) {
	// Test with empty body
	transport := &mockHTTPTransport{
		responses: []mockResponse{
			{status: 200, body: "test-gce-jwt-token"}, // GCE token
			{status: 200, body: ""}, // Empty API response
		},
	}
	storage := &RemoteStorage{
		BaseURL:       "https://api.test.com",
		TokenAudience: "test-audience",
		Client:        &http.Client{Transport: transport},
		Log:           slog.Default(),
	}
	_, _, err := storage.Load("example.com")
	if err == nil {
		t.Fatal("expected error for empty body")
	}

	// Test with malformed JSON
	transport = &mockHTTPTransport{
		responses: []mockResponse{
			{status: 200, body: "test-gce-jwt-token"}, // GCE token
			{status: 200, body: "not json"}, // Malformed API response
		},
	}
	storage.Client = &http.Client{Transport: transport}
	_, _, err = storage.Load("example.com")
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

// TestGCETokenAudience tests GCE token fetching with custom audience
func TestGCETokenAudience(t *testing.T) {
	// Test with custom audience
	transport := &mockHTTPTransport{
		responses: []mockResponse{
			{status: 200, body: "custom-gce-jwt-token"}, // GCE token response
			{status: 200, body: `{"certificate": "test", "metadata": {"expires_at": "2024-12-31T23:59:59Z", "issued_at": "2024-01-01T00:00:00Z"}}`},
		},
	}

	storage := &RemoteStorage{
		BaseURL:       "https://api.test.com",
		TokenAudience: "custom-audience",
		Client:        &http.Client{Transport: transport},
		Log:           slog.Default(),
	}

	_, _, err := storage.Load("example.com")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify GCE token request used correct audience
	if len(transport.requests) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(transport.requests))
	}
	gceReq := transport.requests[0]
	if !strings.Contains(gceReq.URL.String(), "audience=custom-audience") {
		t.Errorf("GCE request should have custom audience in URL: %s", gceReq.URL)
	}
	apiReq := transport.requests[1]
	if apiReq.Header.Get("X-Instance-Token") != "custom-gce-jwt-token" {
		t.Errorf("expected custom GCE token in header")
	}
}

// TestLocalFileWriter tests the local file writing implementation
func TestLocalFileWriter(t *testing.T) {
	writer := &LocalFileWriter{}

	// Test WriteChain
	testChain := ChainPEM("-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----")
	path, err := writer.WriteChain(t.TempDir(), testChain)
	if err != nil {
		t.Fatalf("WriteChain() error = %v", err)
	}
	if !strings.HasSuffix(path, CertFullChainFileName) {
		t.Errorf("WriteChain() path = %s, should end with %s", path, CertFullChainFileName)
	}

	// Verify file contents
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if !bytes.Equal(content, testChain) {
		t.Errorf("file content mismatch")
	}
}
