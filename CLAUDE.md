# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`eigenx` is a CLI toolkit for scaffolding, developing, and testing applications on EigenX. It's built in Go and focuses on deploying docker containers in TEEs onchain.

## Common Development Commands

### Building and Testing
```bash
# Build the CLI binary
make build

# Run all tests (may be slow)
make tests

# Run fast tests (skips slow integration tests) 
make tests-fast

# Install binary to ~/bin/ and set up shell completion
make install

# Format code
make fmt

# Run linter
make lint

# Clean up build artifacts
make clean
```

### Testing the CLI
After building, test the CLI:
```bash
./bin/eigenx --help
./bin/eigenx app --help
```

### Cross-platform Builds
```bash
# Build for specific platforms
make build/darwin-arm64
make build/darwin-amd64  
make build/linux-arm64
make build/linux-amd64

# Build all platforms
make release
```

## Architecture Overview

### CLI Command Structure

The CLI is built with `urfave/cli/v2` and organized hierarchically:
- **Main entry**: `cmd/eigenx/main.go`
- **Core commands**: All under `eigenx app` subcommand
- **Command implementations**: `pkg/commands/` directory

| Command | Description |
| --- | --- |
| `eigenx app create [name] [language]` | Create new app project from template |
| `eigenx app metadata set <app-id\|name>` | Set app metadata (name, website, description, X URL, image) |
| `eigenx app deploy [image_ref]` | Build, push, deploy to TEE |
| `eigenx app upgrade <app-id\|name> <image_ref>` | Upgrade existing deployment |
| `eigenx app start [app-id\|name]` | Start stopped app (start GCP instance) |
| `eigenx app stop [app-id\|name]` | Stop running app (stop GCP instance) |
| `eigenx app terminate [app-id\|name]` | Terminate app (terminate GCP instance) |
| `eigenx app list` | List all deployed apps |
| `eigenx app info [app-id\|name]` | Show detailed app information |
| `eigenx app logs [app-id\|name]` | View logs |

## Interactive Parameters

Optional parameters are requested interactively when not provided:

- **`name`**: Prompts if not provided and not detectable from current directory
- **`language`**: Prompts with selection list if not provided to `create`
- **`app-id|name`**: Prompts with list of available apps if not provided
- **`image_ref`**: Prompts if not provided during deploy/upgrade

### Context Detection

Commands auto-detect project context when run in directory containing `Dockerfile`. Makes `name` parameter optional for: `deploy`.

Commands also support app name resolution - you can use either the full app ID (0x123...) or a friendly name you've set with `eigenx app metadata set`.

### Configuration System
Global configuration with XDG Base Directory compliance:

**Global Config** (`~/.config/eigenx/config.yaml`): User preferences and settings
- `FirstRun`: Tracks if this is the user's first time running eigenx
- `TelemetryEnabled`: User's global telemetry preference  
- `UserUUID`: Unique identifier for telemetry tracking

The global config uses XDG Base Directory specification, falling back to `~/.config` if `XDG_CONFIG_HOME` is not set. Configuration is automatically created with defaults on first run.

### Template System Architecture
Projects are scaffolded from Git-based templates supporting multiple languages (TypeScript, Python, etc.). Templates include:
- Dockerfile and container configuration
- Environment variable setup (.env.example)
- Sample application code
- Build and deployment configuration

### Package Organization
- **`pkg/commands/`**: CLI command implementations
- **`pkg/common/`**: Shared utilities, configuration, contracts, logging
- **`pkg/template/`**: Git-based template management
- **`pkg/telemetry/`**: PostHog analytics integration  
- **`pkg/migration/`**: Configuration migration system
- **`pkg/hooks/`**: Command lifecycle hooks

## Key Dependencies

- **Go 1.23.6+** required
- **Compute TEE contracts**: `github.com/Layr-Labs/eigenx-contracts`
- **Compute TEE KMS**: `github.com/Layr-Labs/eigenx-kms`
- **External tools**: Docker, Foundry

## Development Environment Setup

1. Install prerequisites: Docker, Foundry, Go 1.23.6+, make, jq, yq
2. Clone repository and run `make install`

## Testing Patterns

- Unit tests use standard Go testing
- Integration tests may require Docker and external dependencies
- Use `make tests-fast` for quick feedback during development
- Integration tests in `test/integration/` directory

## Configuration Migration

When adding new configuration fields:
1. Update config structs in `pkg/common/`
2. Create migration in `config/configs/migrations/` or `config/contexts/migrations/`
3. Update embedded config versions in `config/`
4. Test migration with existing project configs

## Telemetry System

Optional PostHog-based telemetry with:
- Global opt-in/opt-out
- Privacy-conscious data collection
- CI environment auto-detection (defaults to disabled)
