# EigenX CLI

**Deploy verifiable applications in Trusted Execution Environments (TEEs)** — Your apps run in secure hardware with their own wallets, enabling autonomous onchain actions while keeping secrets safe.

## Prerequisites

- **Allowlisted Account** - Submit request at [onboarding.eigencloud.xyz](https://onboarding.eigencloud.xyz)
- **Docker** - For packaging applications ([Download](https://www.docker.com/get-started/))
- **Sepolia ETH** - For gas fees ([Faucets](docs/AUTHENTICATION.md#insufficient-funds))

## Quick Start

### 1. Install

```bash
# macOS/Linux
curl -fsSL https://eigenx-scripts.s3.us-east-1.amazonaws.com/install-eigenx.sh | bash

# Windows
curl -fsSL https://eigenx-scripts.s3.us-east-1.amazonaws.com/install-eigenx.ps1 | powershell -
```

→ [More installation options](docs/INSTALLATION.md)

### 2. Authenticate

```bash
# Use an existing key
eigenx auth login

# Or generate a new key and store it
eigenx auth generate --store
```

→ [Authentication guide](docs/AUTHENTICATION.md)

### 3. Deploy Your First App

```bash
# Create from template
eigenx app create my-app typescript
cd my-app

# Configure and deploy
cp .env.example .env
eigenx app deploy
```

### 4. Monitor

```bash
eigenx app info     # View app details
eigenx app logs -w  # Stream logs
```

## Working with Existing Projects

Already have a Docker-based app? Just deploy it:

```bash
cd my-existing-project
eigenx app deploy  # CLI auto-detects Dockerfile and .env
```

**Requirements:**
- Dockerfile targeting `linux/amd64`
- Application runs as root user
- Optional: `.env` file for configuration

→ [TLS/HTTPS Setup](docs/TLS_SETUP.md) | [Command Reference](docs/COMMANDS.md)

## Key Features

### 🔒 Secure Execution
Your code runs in Intel TDX instances with hardware-level isolation

### 🔑 Auto-Generated Wallets
Each app gets a unique mnemonic accessible via `process.env.MNEMONIC`

### 🌍 Environment Variables
- Standard vars encrypted and private to your TEE
- `_PUBLIC` suffix for transparent variables

### ⛓️ Onchain Management
App lifecycle controlled via Ethereum smart contracts

## Common Commands

```bash
eigenx app list              # List your apps
eigenx app deploy            # Deploy app
eigenx app stop my-app       # Stop an app
eigenx app start my-app      # Start an app
eigenx app upgrade my-app    # Deploy new version
eigenx app terminate my-app  # Remove permanently
```

→ [Full command reference](docs/COMMANDS.md)

## Documentation

| Guide | Description |
|-------|-------------|
| [Installation](docs/INSTALLATION.md) | All installation methods and troubleshooting |
| [Authentication](docs/AUTHENTICATION.md) | Private key management and security |
| [Commands](docs/COMMANDS.md) | Complete CLI command reference |
| [TLS Setup](docs/TLS_SETUP.md) | Enable HTTPS with automatic certificates |
| [Core Concepts](docs/CONCEPTS.md) | Keys, privacy, security, and app lifecycle |
| [Architecture](docs/ARCHITECTURE.md) | Technical deep-dive into how EigenX works |
| [Development](docs/DEVELOPMENT.md) | Build from source and contribute |
| [Telemetry](docs/TELEMETRY.md) | Usage analytics and privacy |

## Mainnet Alpha Status

⚠️ **Current Limitations:**
- **Not recommended for customer funds** - Mainnet Alpha is intended to enable developers to build, test and ship applications. We do not recommend holding significant customer funds at this stage in Mainnet Alpha.
- **Developer is still trusted** - Mainnet Alpha does not enable full verifiable and trustless execution. A later version will ensure developers can not upgrade code maliciously, and liveness guarantees.
- **No SLA** - Mainnet Alpha does not have SLAs around support, and uptime of infrastructure.

## Support

- **Issues**: [GitHub Issues](https://github.com/Layr-Labs/eigenx-cli/issues)
- **Security**: Report vulnerabilities to `security@eigenlabs.org`

## License

MIT - See [LICENSE](LICENSE) file

---

**🚧 EigenX is in alpha** - Features may change, interfaces will evolve. Use for testing and development only. Provided "as is" without guarantee of functionality or production support.
