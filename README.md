# EigenX CLI

**Deploy Docker containers in Trusted Execution Environments (TEEs) via smart contracts**

EigenX lets you deploy containerized applications that run in secure, verifiable compute environments with built-in private key management. Your apps get a unique wallet they control, enabling autonomous onchain actions while keeping secrets safe.

## Prerequisites

- **Allowlisted Account** - Required to create apps. Use existing address or generate with `eigenx auth generate`. Submit to Eigen team for allowlisting.
- **Docker** - To package and publish application images ([Download](https://www.docker.com/get-started/))
- **Sepolia ETH** - For deployment transactions ([Google Cloud Faucet](https://cloud.google.com/application/web3/faucet/ethereum/sepolia) | [Alchemy Faucet](https://sepoliafaucet.com/))

## Mainnet Alpha Limitations
- ** Not recommended for customer funds ** - Mainnet Alpha is intended to enable developers to build, test and ship applications. We do not recommend holding significant customer funds at this stage in Mainnet Alpha.
- ** Developer is still trusted ** - Mainnet Alpha does not enable full verifiable and trustless execution. A later version will ensure developers can not upgrade code maliciously, and liveness gaurantees.
- ** No SLA ** - Mainnet Alpha does not have SLAs around support, and uptime of infrastructure. 

## **Quick Start**

### **Installation**

#### macOS/Linux
```bash
curl -fsSL https://eigenx-scripts.s3.us-east-1.amazonaws.com/install-eigenx.sh | bash
```

#### Windows
```bash
curl -fsSL https://eigenx-scripts.s3.us-east-1.amazonaws.com/install-eigenx.ps1 | powershell -
```

### Setup (One-time)
```bash
# Log in to your Docker registry (required to push images)
docker login

# Generate and store your private key
eigenx auth generate --store
```

**Already have a private key?** Use `eigenx auth login` instead

**Need Sepolia ETH?** Run `eigenx auth whoami` to see your address, then get funds from [Google Cloud](https://cloud.google.com/application/web3/faucet/ethereum/sepolia) or [Alchemy](https://sepoliafaucet.com/)

### **Create & Deploy**

```bash
# Create your app (choose: typescript | python | golang | rust)
eigenx app create my-app typescript
cd my-app

# Configure environment variables
cp .env.example .env

# Deploy to TEE
eigenx app deploy
```

### **Working with Existing Projects**

Have an existing project? You don't need `eigenx app create` - the CLI works with any Docker-based project:

```bash
# From your existing project directory
cd my-existing-project

# Ensure you have a Dockerfile and .env file
# The CLI will prompt for these if not found in standard locations

# Deploy directly - the CLI will detect your project
eigenx app deploy
```

**What you need:**
- **Dockerfile** - Must target `linux/amd64` and run as root user
- **.env file** - For environment variables (optional but recommended)

The CLI will automatically prompt for the Dockerfile and .env paths if they're not in the default locations. This means you can use eigenx with any existing containerized application without restructuring your project.

**Need TLS/HTTPS?** Run `eigenx app configure tls` to add the necessary configuration files for domain setup with private traffic termination in the TEE.

### **View Your App**

```bash
# View app information
eigenx app info

# View app logs
eigenx app logs
```

That's it! Your starter app is now running in a TEE with access to a MNEMONIC that only it can access.

**Ready to customize?** Edit your application code, update `.env` with any API keys you need, then run `eigenx app upgrade my-app` to deploy your changes

## Application Environment

Your TEE application runs with these capabilities:

1. **Secure Execution** - Your code runs in an Intel TDX instance with hardware-level isolation
2. **Auto-Generated Wallet** - Access a private mnemonic via `process.env.MNEMONIC`
    - Derive wallet accounts using standard libraries (e.g., viem’s `mnemonicToAccount(process.env.MNEMONIC)`)
    - Only your TEE can decrypt and use this mnemonic
3. **Environment Variables** - All variables from your `.env` file are available in your container
   - Variables with `_PUBLIC` suffix are visible to users for transparency
   - Standard variables remain private and encrypted within the TEE
4. **Onchain Management** - Your app's lifecycle is controlled via Ethereum smart contracts

### Working with Your App

```bash
# List all your apps
eigenx app list

# Stop/start your app
eigenx app stop my-app
eigenx app start my-app

# Terminate your app
eigenx app terminate my-app
```

## Authentication

EigenX CLI needs a private key to sign transactions. Three options:

### 1. OS Keyring (Recommended)

```bash
eigenx auth generate --store # Generate new key and store it
eigenx auth login            # Store an existing key securely
eigenx auth whoami           # Check authentication
eigenx auth logout           # Remove key
```

### 2. Environment Variable

```bash
export PRIVATE_KEY=0x1234...
eigenx app deploy
```

### 3. Command Flag

```bash
eigenx app deploy --private-key 0x1234...
```

**Priority:** Flag → Environment → Keyring

## TLS/HTTPS Setup

### Enable TLS

```bash
# Add TLS configuration to your project
eigenx app configure tls

# Add variables to .env
cat .env.example.tls >> .env
```

### Configure

Required in `.env`:
```bash
DOMAIN=yourdomain.com
APP_PORT=3000
```

Recommended for first deployment:
```bash
ENABLE_CADDY_LOGS=true  # Debug logs
ACME_STAGING=true       # Test certificates (avoid rate limits)
```

### DNS Setup

Create A record pointing to instance IP:
- Type: A
- Name: yourdomain.com
- Value: `<instance-ip>` (get from `eigenx app info`)

### Deploy

```bash
eigenx app upgrade
```

### Production Certificates

To switch from staging to production:
```bash
# Set in .env:
ACME_STAGING=false
ACME_FORCE_ISSUE=true  # Only if staging cert exists

# Deploy, then set ACME_FORCE_ISSUE=false for future deploys
```

**Notes:**
- Let's Encrypt rate limit: 5 certificates/week per domain
- Test with staging certificates first to avoid rate limits
- DNS changes may take a few minutes to propagate

## Complete Command Reference

### Authentication

| Command | Description |
| --- | --- |
| `eigenx auth generate` | Generate new private key and optionally store it (aliases: `gen`, `new`) |
| `eigenx auth login` | Store existing private key in OS keyring |
| `eigenx auth whoami` | Show current authentication status and address |
| `eigenx auth list` | List all stored private keys by environment |
| `eigenx auth logout` | Remove private key from OS keyring |

### Project Management

| Command | Description |
| --- | --- |
| `eigenx app create [name] [language]` | Create new project from template |
| `eigenx app configure tls` | Add TLS configuration to your project |
| `eigenx app name <app-id\|name> <new-name>` | Set a friendly name for your app |

### Deployment & Updates

| Command | Description |
| --- | --- |
| `eigenx app deploy [image_ref]` | Deploy new app to TEE |
| `eigenx app upgrade <app-id\|name> <image_ref>` | Update existing deployment |

### Lifecycle Management

| Command | Description |
| --- | --- |
| `eigenx app start [app-id\|name]` | Start stopped app |
| `eigenx app stop [app-id\|name]` | Stop running app |
| `eigenx app terminate [app-id\|name]` | Permanently remove app |

### Monitoring

| Command | Description |
| --- | --- |
| `eigenx app list` | List all your deployed apps |
| `eigenx app info [app-id\|name]` | Show detailed app information |
| `eigenx app logs [app-id\|name]` | View application logs |

### Deployment Environment Management

| Command | Description |
| --- | --- |
| `eigenx environment show` | Show active deployment environment (alias: `env`) |
| `eigenx environment list` | List available deployment environments |
| `eigenx environment set <environment>` | Set deployment environment |

### Configuration

| Command | Description |
| --- | --- |
| `eigenx telemetry [--enable\|--disable\|--status]` | Manage usage analytics |
| `eigenx upgrade` | Update CLI to latest version |
| `eigenx version` | Show CLI version |

## Advanced Usage

### Building and Pushing Images Manually

If you prefer to build and push Docker images yourself instead of letting the CLI handle it, or already have an existing image:

```bash
# Build and push your image manually
docker build --platform linux/amd64 -t myregistry/myapp:v1.0 .
docker push myregistry/myapp:v1.0

# Deploy using the image reference
eigenx app deploy myregistry/myapp:v1.0
```

**Requirements:**

- Image must target `linux/amd64` architecture
- Application must run as root user (TEE requirement)

## Telemetry

EigenX collects anonymous usage data to help us improve the CLI and understand how it's being used. This telemetry is enabled by default but can be easily disabled.

### What We Collect

- Commands used (e.g., `eigenx app create`, `eigenx app deploy`)
- Error counts and types to identify common issues
- Performance metrics (command execution times)
- System information (OS, architecture)
- Geographic location (country/city level only)

### What We DON'T Collect

- Personal information or identifiers
- Private keys or sensitive credentials
- Application source code or configurations
- Specific file paths or project names

### Managing Telemetry

```bash
# Check current telemetry status
eigenx telemetry --status

# Disable telemetry
eigenx telemetry --disable

# Re-enable telemetry
eigenx telemetry --enable
```

Telemetry settings are stored globally and persist across all projects.

## Architecture Overview

### How TEEs Work

- **Hardware Isolation** - Secure enclaves separated from host OS
- **Attestation** - Cryptographic proof of code integrity
- **Secure Key Access** - Private keys via Google KMS, accessible only to verified TEEs

### Smart Contract Integration

- Register and verify TEE deployments
- Manage app lifecycle and updates
- Coordinate developers and infrastructure

## Development

Build from source:

```bash
git clone https://github.com/Layr-Labs/eigenx-cli
cd eigenx-cli
make build              # Builds for development (default)
GO_TAGS=prod make build # Builds for production
./bin/eigenx --help
```

Run tests:
```bash
make tests        # Full test suite
make tests-fast   # Quick tests only
```

## Release Process

### Local Template Development

For testing template changes locally without pushing to GitHub:

1. Clone the templates repository:
```bash
git clone https://github.com/Layr-Labs/eigenx-templates
```

2. Use local templates with environment variables:
```bash
# From the eigenx-cli directory or one level above
EIGENX_USE_LOCAL_TEMPLATES=true ./bin/eigenx app create test-app typescript

# Or specify the path explicitly
EIGENX_TEMPLATES_PATH=/path/to/eigenx-templates EIGENX_USE_LOCAL_TEMPLATES=true eigenx app create test-app golang
```

This uses templates from your local `eigenx-templates/` directory instead of fetching from GitHub.

### Dev Releases

For testing new features before production:

```bash
# Tag with dev suffix and incremental build number
git tag v0.1.0-dev.1
git push origin v0.1.0-dev.1

# Found a bug? Increment the build number
git tag v0.1.0-dev.2
git push origin v0.1.0-dev.2
```

This deploys to the dev environment and can be installed with:
```bash
curl -fsSL https://eigenx-scripts.s3.us-east-1.amazonaws.com/install-eigenx.sh | bash -s -- --dev
```

### Production Releases

After testing in dev, promote to production:

```bash
# Promotes the latest v0.1.0-dev.* version to production
git tag v0.1.0
git push origin v0.1.0
```

This:
- Verifies a dev version exists (e.g., `v0.1.0-dev.2`)
- Copies the **exact same binary** from dev to production
- Creates a GitHub release
- Updates the stable installation channel

Users get the promoted version with:
```bash
curl -fsSL https://eigenx-scripts.s3.us-east-1.amazonaws.com/install-eigenx.sh | bash
```

### Key Benefits

- **Build once**: Same binary promoted from dev to prod
- **Safety**: Cannot release untested code to production
- **Coexistence**: `eigenx-dev` and `eigenx` can both be installed
- **Clean versions**: Production gets clean version numbers (v0.1.0)
- **Easy rollback**: Tag any previous version for instant rollback

### Installation Channels

- **Development**: `--dev` flag installs as `eigenx-dev`
- **Production**: Default installation installs as `eigenx`

Both versions can coexist on the same system.

## Disclaimer
🚧 eigenx-cli is under active development and has not been audited. eigenx-cli is rapidly being upgraded, features may be added, removed or otherwise improved or modified and interfaces will have breaking changes. eigenx-cli should be used only for testing purposes and not in production. eigenx-cli is provided "as is" and Eigen Labs, Inc. does not guarantee its functionality or provide support for its use in production. 🚧
