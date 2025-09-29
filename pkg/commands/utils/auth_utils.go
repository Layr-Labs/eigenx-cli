package utils

import (
	"fmt"
	"math/big"

	"github.com/urfave/cli/v2"
)

// GenerateAuthHeaders generates Authorization and X-eigenx-expiry headers for API authentication
// by using the contract to calculate and sign the permission digest
func GenerateAuthHeaders(
	cCtx *cli.Context,
	permission [4]byte,
	expiry *big.Int,
) (map[string]string, error) {
	// Calculate and sign the API permission digest using the contract
	signature, err := CalculateAndSignApiPermissionDigest(cCtx, permission, expiry)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate and sign digest: %w", err)
	}

	// Create headers
	headers := map[string]string{
		"Authorization":   fmt.Sprintf("Bearer %x", signature),
		"X-eigenx-expiry": expiry.String(),
	}

	return headers, nil
}
