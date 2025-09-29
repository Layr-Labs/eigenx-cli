package telemetry

import (
	"context"
	"os"

	"github.com/Layr-Labs/eigenx-cli/pkg/common"

	"github.com/posthog/posthog-go"
)

// PostHogClient implements the Client interface using PostHog
type PostHogClient struct {
	namespace      string
	client         posthog.Client
	appEnvironment *common.AppEnvironment
}

// NewPostHogClient creates a new PostHog client
func NewPostHogClient(environment *common.AppEnvironment, namespace string) (*PostHogClient, error) {
	apiKey := getPostHogAPIKey()
	if apiKey == "" {
		// No API key available, return noop client without error
		return nil, nil
	}
	client, err := posthog.NewWithConfig(apiKey, posthog.Config{Endpoint: getPostHogEndpoint()})
	if err != nil {
		return nil, err
	}
	return &PostHogClient{
		namespace:      namespace,
		client:         client,
		appEnvironment: environment,
	}, nil
}

// AddMetric implements the Client interface
func (c *PostHogClient) AddMetric(_ context.Context, metric Metric) error {
	if c == nil || c.client == nil {
		return nil
	}

	// Create properties map starting with base properties
	props := make(map[string]interface{})
	// Add metric value
	props["name"] = metric.Name
	props["value"] = metric.Value

	// Add metric dimensions
	for k, v := range metric.Dimensions {
		props[k] = v
	}

	// Never return errors from telemetry operations
	err := c.client.Enqueue(posthog.Capture{
		DistinctId: c.appEnvironment.UserUUID,
		Event:      c.namespace,
		Properties: props,
	})
	return err
}

// Close implements the Client interface
func (c *PostHogClient) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	// Ignore any errors from Close operations
	_ = c.client.Close()
	return nil
}

func getPostHogAPIKey() string {
	// Priority order:
	// 1. Environment variable
	// 2. Embedded key (set at build time)
	// Check environment variable first
	if key := os.Getenv("EIGENX_POSTHOG_KEY"); key != "" {
		return key
	}

	// return embedded key if no overrides provided
	return embeddedTelemetryApiKey
}

func getPostHogEndpoint() string {
	if endpoint := os.Getenv("EIGENX_POSTHOG_ENDPOINT"); endpoint != "" {
		return endpoint
	}
	return "https://us.i.posthog.com"
}
