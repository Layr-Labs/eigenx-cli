# Development Guide

This guide covers building, testing, and contributing to the EigenX CLI.

## Prerequisites

### Required Tools
- **Go 1.23.6+** - [Download](https://golang.org/dl/)
- **Make** - Build automation
- **Docker** - For testing deployments
- **Git** - Version control

### Optional Tools
- **Foundry** - For smart contract interaction
- **jq** - JSON processing in scripts
- **yq** - YAML processing

## Building from Source

### Quick Build

```bash
# Clone repository
git clone https://github.com/Layr-Labs/eigenx-cli
cd eigenx-cli

# Build for development (default)
make build

# Run the built binary
./bin/eigenx --help
```

### Build Targets

```bash
# Development build (includes debug info)
make build

# Production build (optimized, no debug)
GO_TAGS=prod make build

# Install to ~/bin
make install

# Build for specific platforms
make build/darwin-arm64    # macOS Apple Silicon
make build/darwin-amd64    # macOS Intel
make build/linux-amd64     # Linux x64
make build/linux-arm64     # Linux ARM64
make build/windows-amd64   # Windows x64

# Build all platforms
make release
```

### Build Options

```bash
# Custom output directory
make build OUTPUT_DIR=/custom/path

# Verbose build
make build VERBOSE=1

# Build with race detector (development only)
make build GOFLAGS="-race"
```

## Testing

### Running Tests

```bash
# Run all tests
make tests

# Run fast tests only (skip integration)
make tests-fast

# Run specific test
go test ./pkg/commands -run TestDeploy

# Run with coverage
make test-coverage

# Run with verbose output
go test -v ./...
```

### Test Organization

```
eigenx-cli/
├── pkg/
│   ├── commands/      # Command implementation tests
│   ├── common/        # Utility tests
│   └── template/      # Template system tests
└── test/
    ├── integration/   # Integration tests
    └── e2e/          # End-to-end tests (coming soon)
```

### Writing Tests

```go
// pkg/commands/deploy_test.go
func TestDeployCommand(t *testing.T) {
    // Setup
    app := cli.NewApp()

    // Test
    err := app.Run([]string{"eigenx", "app", "deploy"})

    // Assert
    assert.NoError(t, err)
}
```

### Integration Tests

Integration tests require Docker and network access:

```bash
# Run integration tests
make test-integration

# Skip integration tests
go test -short ./...
```

## Development Workflow

### 1. Create Feature Branch

```bash
git checkout -b feature/my-feature
```

### 2. Make Changes

```bash
# Edit code
vim pkg/commands/my-command.go

# Format code
make fmt

# Run linter
make lint
```

### 3. Test Changes

```bash
# Run tests
make tests-fast

# Manual testing
./bin/eigenx app deploy --help
```

### 4. Commit Changes

```bash
git add .
git commit -m "feat: add new command"
```

## Code Organization

### Project Structure

```
eigenx-cli/
├── cmd/
│   └── eigenx/          # Main entry point
├── pkg/
│   ├── commands/        # CLI command implementations
│   ├── common/          # Shared utilities
│   │   ├── config/      # Configuration management
│   │   ├── contracts/   # Smart contract ABIs
│   │   └── logging/     # Logging utilities
│   ├── template/        # Project template system
│   ├── telemetry/       # Analytics
│   └── migration/       # Config migration
├── scripts/             # Build and release scripts
├── docs/               # Documentation
└── test/               # Test files
```

### Key Packages

| Package | Description |
|---------|-------------|
| `pkg/commands` | All CLI commands |
| `pkg/common/config` | Configuration management |
| `pkg/common/contracts` | Smart contract interaction |
| `pkg/template` | Template system for project creation |
| `pkg/telemetry` | PostHog telemetry integration |

## Configuration System

### Global Configuration

Location: `~/.config/eigenx/config.yaml`

```yaml
FirstRun: false
TelemetryEnabled: true
UserUUID: "uuid-here"
Environment: "sepolia"
```

### XDG Compliance

The CLI follows XDG Base Directory spec:

```bash
# Override config location
export XDG_CONFIG_HOME=/custom/config
eigenx app list
```

## Template Development

### Local Template Testing

```bash
# Clone templates repository
git clone https://github.com/Layr-Labs/eigenx-templates ../eigenx-templates

# Use local templates
EIGENX_USE_LOCAL_TEMPLATES=true ./bin/eigenx app create test-app typescript

# Or specify path
EIGENX_TEMPLATES_PATH=/path/to/templates EIGENX_USE_LOCAL_TEMPLATES=true \
  eigenx app create test-app
```

### Creating New Templates

1. Fork [eigenx-templates](https://github.com/Layr-Labs/eigenx-templates)
2. Create new directory: `templates/my-language/`
3. Add required files:
   - `Dockerfile`
   - `.env.example`
   - Application code
4. Test locally
5. Submit PR

## Release Process

### Version Tagging

We use semantic versioning with a two-stage release process:

### Development Releases

```bash
# Create dev release
git tag v0.1.0-dev.1
git push origin v0.1.0-dev.1

# Incremental dev releases
git tag v0.1.0-dev.2
git push origin v0.1.0-dev.2
```

Install dev version:
```bash
curl -fsSL https://eigenx-scripts.s3.us-east-1.amazonaws.com/install-eigenx.sh | \
  bash -s -- --dev
```

### Production Releases

```bash
# Promote dev to production
git tag v0.1.0
git push origin v0.1.0
```

This:
- Verifies dev version exists
- Copies exact binary to production
- Creates GitHub release
- Updates stable channel

### Release Automation

GitHub Actions handles:
1. Building binaries for all platforms
2. Uploading to S3
3. Creating GitHub releases
4. Updating installation scripts

## Debugging

### Debug Mode

```bash
# Enable debug logging
EIGENX_DEBUG=true ./bin/eigenx app deploy

# Trace mode (very verbose)
EIGENX_TRACE=true ./bin/eigenx app deploy
```

### Common Issues

#### Build Fails

```bash
# Clean and rebuild
make clean
make build

# Check Go version
go version  # Should be 1.23.6+
```

#### Tests Fail

```bash
# Update dependencies
go mod download
go mod tidy

# Clear test cache
go clean -testcache
```

## Contributing

### Before Submitting PR

1. **Format code:**
   ```bash
   make fmt
   ```

2. **Run linter:**
   ```bash
   make lint
   ```

3. **Run tests:**
   ```bash
   make tests
   ```

4. **Update documentation:**
   - Add command to [COMMANDS.md](COMMANDS.md)
   - Update README if needed

### PR Guidelines

- Use conventional commits:
  - `feat:` New features
  - `fix:` Bug fixes
  - `docs:` Documentation
  - `test:` Tests
  - `refactor:` Code refactoring
  - `chore:` Maintenance

- Include tests for new features
- Update documentation
- Keep PRs focused and small

### Code Style

- Follow Go conventions
- Use meaningful variable names
- Add comments for complex logic
- Keep functions small and focused

## Advanced Topics

### Smart Contract Integration

Contract ABIs are embedded in:
```
pkg/common/contracts/
├── abi/
│   ├── Coordinator.json
│   └── ComputeTEE.json
└── contracts.go
```

Update process:
1. Get new ABIs from [eigenx-contracts](https://github.com/Layr-Labs/eigenx-contracts)
2. Update JSON files
3. Run `go generate`

### Adding New Commands

1. Create command file:
   ```go
   // pkg/commands/mycommand.go
   package commands

   func MyCommand() *cli.Command {
       return &cli.Command{
           Name:   "mycommand",
           Usage:  "Description here",
           Action: myCommandAction,
       }
   }
   ```

2. Register in main:
   ```go
   // cmd/eigenx/main.go
   app.Commands = append(app.Commands, commands.MyCommand())
   ```

3. Add tests:
   ```go
   // pkg/commands/mycommand_test.go
   func TestMyCommand(t *testing.T) {
       // Test implementation
   }
   ```

### Telemetry Integration

Add telemetry to new features:

```go
import "github.com/Layr-Labs/eigenx-cli/pkg/telemetry"

func myCommandAction(c *cli.Context) error {
    // Track command usage
    telemetry.Track("my_command_used", map[string]interface{}{
        "option": c.String("option"),
    })

    // Command logic
    return nil
}
```

## Useful Make Commands

```bash
# Development
make build          # Build binary
make install        # Install binary
make fmt           # Format code
make lint          # Run linter

# Testing
make tests         # All tests
make tests-fast    # Quick tests
make test-coverage # Coverage report

# Release
make release       # Build all platforms
make clean        # Clean build artifacts

# Docker
make docker-build  # Build Docker image
make docker-push   # Push to registry
```

## Environment Variables

### Development Variables

| Variable | Description |
|----------|-------------|
| `EIGENX_DEBUG` | Enable debug logging |
| `EIGENX_TRACE` | Enable trace logging |
| `EIGENX_USE_LOCAL_TEMPLATES` | Use local template directory |
| `EIGENX_TEMPLATES_PATH` | Custom templates path |
| `EIGENX_TELEMETRY_DEBUG` | Debug telemetry events |
| `EIGENX_TELEMETRY_DISABLED` | Disable telemetry |

### Build Variables

| Variable | Description |
|----------|-------------|
| `GO_TAGS` | Build tags (e.g., `prod`) |
| `OUTPUT_DIR` | Custom output directory |
| `VERBOSE` | Verbose build output |

## Getting Help

- [GitHub Issues](https://github.com/Layr-Labs/eigenx-cli/issues) - Bug reports and features
- [Discord](https://discord.gg/eigencloud) - Community support
- [Documentation](https://github.com/Layr-Labs/eigenx-cli/docs) - Full documentation

## License

MIT License - See [LICENSE](../LICENSE) file