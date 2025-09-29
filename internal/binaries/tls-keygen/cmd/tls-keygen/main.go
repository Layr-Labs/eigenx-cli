package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Layr-Labs/eigenx-cli/internal/binaries/tls-keygen/cert"
	"github.com/Layr-Labs/eigenx-cli/internal/binaries/tls-keygen/config"
	"github.com/Layr-Labs/eigenx-cli/internal/binaries/tls-keygen/storage"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "tls-keygen",
		Usage: "ACME certificate management for TEE deployments",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "mnemonic",
				Usage:    "BIP39 mnemonic phrase (required)",
				EnvVars:  []string{"MNEMONIC"},
				Required: false, // We'll validate this manually for better error messages
			},
			&cli.StringFlag{
				Name:     "domain",
				Usage:    "Domain name for TLS certificate (required)",
				EnvVars:  []string{"DOMAIN"},
				Required: false, // We'll validate this manually
			},
			&cli.StringFlag{
				Name:    "email",
				Usage:   "Contact email for ACME account",
				EnvVars: []string{"ACME_EMAIL"},
			},
			&cli.StringFlag{
				Name:    "challenge",
				Usage:   "Challenge type: http-01 or tls-alpn-01",
				Value:   "http-01",
				EnvVars: []string{"ACME_CHALLENGE"},
			},
			&cli.StringFlag{
				Name:    "ca",
				Usage:   "ACME CA URL (overrides -staging)",
				EnvVars: []string{"ACME_CA"},
			},
			&cli.BoolFlag{
				Name:    "staging",
				Usage:   "Use Let's Encrypt staging environment",
				EnvVars: []string{"ACME_STAGING"},
			},
			&cli.DurationFlag{
				Name:    "timeout",
				Usage:   "Overall timeout",
				Value:   2 * time.Minute,
				EnvVars: []string{"ACME_TIMEOUT"},
			},
			&cli.BoolFlag{
				Name:    "force",
				Usage:   "Force certificate reissue even if a valid one exists",
				EnvVars: []string{"ACME_FORCE_ISSUE"},
			},
			&cli.DurationFlag{
				Name:    "renewal-window",
				Usage:   "Renew if cert expires within this window",
				Value:   30 * 24 * time.Hour,
				EnvVars: []string{"ACME_RENEWAL_WINDOW"},
			},
			&cli.UintFlag{
				Name:    "version",
				Usage:   "Deterministic TLS key version (rotations)",
				Value:   0,
				EnvVars: []string{"TLS_KEY_VERSION"},
			},
			&cli.StringFlag{
				Name:    "api-url",
				Usage:   "Certificate store service URL",
				EnvVars: []string{"API_URL"},
			},
			&cli.StringFlag{
				Name:    "alt-names",
				Usage:   "Additional SANs (comma-separated)",
				EnvVars: []string{"ALT_NAMES"},
			},
			&cli.StringFlag{
				Name:    "token-audience",
				Usage:   "Audience for GCE identity tokens",
				Value:   "eigen-cert-storage",
				EnvVars: []string{"INSTANCE_TOKEN_AUDIENCE"},
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) error {
	// Build options from CLI context
	opts := buildOptions(c)

	// Validate configuration
	if err := opts.Validate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create remote storage (required)
	if opts.APIURL == "" {
		return fmt.Errorf("api-url is required for certificate storage")
	}

	remoteStorage := storage.NewRemoteStorage(opts.APIURL, opts.TokenAudience, logger)

	// Create local writer
	localWriter := storage.LocalFileWriter{}

	// Create certificate manager using Lego
	certManager := cert.NewLegoManager(remoteStorage, localWriter, logger)

	// Run with timeout
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	// Ensure certificate
	result, err := certManager.EnsureCertificate(ctx, opts)
	if err != nil {
		return fmt.Errorf("error obtaining certificate: %w", err)
	}

	// Log result
	if result.Issued {
		logger.Info("issued new cert",
			"domain", opts.Domain,
			"not_after", result.NotAfter.Format(time.RFC3339),
			"days", int(time.Until(result.NotAfter).Hours()/24),
			"cert", result.FullChainPath,
			"key", result.PrivKeyPath)
	} else {
		logger.Info("using existing cert",
			"domain", opts.Domain,
			"days_left", int(time.Until(result.NotAfter).Hours()/24),
			"cert", result.FullChainPath,
			"key", result.PrivKeyPath,
			"reconstructed", result.Reconstructed)
	}

	return nil
}

// buildOptions constructs Options from CLI context
func buildOptions(c *cli.Context) config.Config {
	// Get mnemonic
	mnemonic := c.String("mnemonic")
	if mnemonic == "" {
		log.Fatal("Error: mnemonic is required (via --mnemonic flag or MNEMONIC env var)")
	}

	// Get domain
	domain := c.String("domain")
	if domain == "" || domain == "localhost" {
		log.Fatal("Error: domain is required and cannot be localhost")
	}

	// Check for force issue
	forceIssue := c.Bool("force")

	// Parse alt names
	var altNamesList []string
	if altNames := c.String("alt-names"); altNames != "" {
		altNamesList = strings.Split(altNames, ",")
		for i := range altNamesList {
			altNamesList[i] = strings.TrimSpace(altNamesList[i])
		}
	}

	// Determine CA URL
	caURL := c.String("ca")
	staging := c.Bool("staging")
	if caURL == "" {
		if staging {
			caURL = config.LEStaging
		} else {
			caURL = config.LEProd
		}
	}

	// Get token audience with default
	tokenAudience := c.String("token-audience")
	if tokenAudience == "" {
		tokenAudience = "eigen-cert-storage"
	}

	return config.Config{
		Mnemonic:      mnemonic,
		Domain:        domain,
		AltNames:      altNamesList,
		Email:         c.String("email"),
		OutDir:        "/run/tls", // Hardcoded
		Challenge:     config.Challenge(c.String("challenge")),
		CADir:         caURL,
		Timeout:       c.Duration("timeout"),
		RenewalWindow: c.Duration("renewal-window"),
		Version:       uint32(c.Uint("version")),
		ForceIssue:    forceIssue,
		APIURL:        c.String("api-url"),
		Staging:       staging,
		TokenAudience: tokenAudience,
		UserAgent:     "eigenx-tls-keygen/1.0",
	}
}