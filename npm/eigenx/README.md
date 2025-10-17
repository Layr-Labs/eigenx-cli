# @layr-labs/eigenx

**Deploy verifiable applications in Trusted Execution Environments (TEEs)**

## Installation

### Global Install (Recommended)

```bash
npm install -g @layr-labs/eigenx
```

After installation, the `eigenx` command will be available globally.

### Using npx (No Install Required)

Run eigenx commands without installing:

```bash
# Run latest version
npx @layr-labs/eigenx app list

# Pin to specific version
npx @layr-labs/eigenx@1.2.3 app deploy

# Use dev version
npx @layr-labs/eigenx@dev app create
```

**Note:** The first `npx` run downloads the package (~10-20MB). Subsequent runs use the cached version and are much faster. For frequent use, global installation is recommended.

## Usage

```bash
# Get help
eigenx --help

# Create a new app
eigenx app create

# Deploy an app
eigenx app deploy

# List deployed apps
eigenx app list

# View app information
eigenx app info

# View logs
eigenx app logs

# Start/stop/terminate apps
eigenx app start
eigenx app stop
eigenx app terminate

# Manage app names
eigenx app name
```

## Supported Platforms

eigenx provides pre-built binaries for the following platforms:

- **macOS**: arm64 (Apple Silicon), x64 (Intel)
- **Linux**: arm64, x64
- **Windows**: x64, arm64

The correct binary for your platform will be automatically installed.

## Requirements

- Node.js 14.0.0 or higher (for installation via npm)
- Docker (for building and deploying apps)

## Documentation

For detailed documentation, visit the [eigenx-cli repository](https://github.com/Layr-Labs/eigenx-cli).

## Issues

If you encounter any problems, please [open an issue](https://github.com/Layr-Labs/eigenx-cli/issues).

## License

MIT
