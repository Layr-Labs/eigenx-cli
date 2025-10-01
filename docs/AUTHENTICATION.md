# Authentication Guide

This guide explains how to authenticate with the EigenX CLI and manage your private keys securely.

## Overview

The EigenX CLI requires a private key to:
- Sign deployment transactions on Ethereum
- Authenticate with the EigenX smart contracts
- Manage your deployed applications

**Important:** This is your developer authentication key, NOT the application's TEE mnemonic. See [Key Types](#key-types) for details.

## Key Types

### Developer Authentication Key
- **Purpose:** Authenticate you as a developer
- **Storage:** Your local machine (OS keyring or environment)
- **Usage:** Sign transactions, deploy apps
- **Access:** Only you have access

### TEE Application Mnemonic
- **Purpose:** Your app's wallet for onchain actions
- **Storage:** KMS (Key Management Service)
- **Usage:** Available as `process.env.MNEMONIC` in your app
- **Access:** Only your TEE can decrypt and use

## Authentication Methods

The CLI checks for authentication in this order:
1. Command-line flag (`--private-key`)
2. Environment variable (`PRIVATE_KEY`)
3. OS keyring (recommended)

### Method 1: OS Keyring (Recommended)

Most secure method - stores key encrypted in your OS keyring.

#### Store a New Key
```bash
# Generate and store a new key
eigenx auth generate --store

# You'll see:
# Generated new private key
# Address: 0x742d35Cc6634C053...
# Private key stored in keyring for environment: sepolia
```

#### Store an Existing Key
```bash
# Interactive - prompts for key
eigenx auth login

# Or provide directly (be careful with shell history!)
echo "your-private-key" | eigenx auth login
```

#### Check Authentication
```bash
eigenx auth whoami

# Output:
# Environment: sepolia
# Address: 0x742d35Cc6634C053...
# Source: keyring
```

#### Remove Key
```bash
# Remove for current environment
eigenx auth logout

# Remove all stored keys
eigenx auth logout --all
```

### Method 2: Environment Variable

Good for CI/CD and automation.

```bash
export PRIVATE_KEY=0x1234...
eigenx app deploy
```

**Security Tips:**
- Never commit `.env` files with private keys
- Use secrets management in CI/CD
- Rotate keys regularly

### Method 3: Command Flag

Use for one-off commands (not recommended for regular use).

```bash
eigenx app deploy --private-key 0x1234...
```

**Warning:** This may be saved in shell history!

## Managing Multiple Environments

The CLI supports different keys for different environments.

### List All Keys
```bash
eigenx auth list

# Output:
# Stored keys:
#   sepolia: 0x742d35Cc6634C053...
#   mainnet-alpha: 0x8B3a350cf5B3C456...
```

### Switch Environment
```bash
# Set environment
eigenx environment set mainnet-alpha

# Now uses mainnet-alpha key
eigenx auth whoami
```

### Environment-Specific Login
```bash
# Login for specific environment
eigenx environment set mainnet-alpha
eigenx auth login
```

## Security Best Practices

### DO's

1. **Use OS Keyring for Development**
   ```bash
   eigenx auth login
   ```

2. **Use Environment Variables in CI/CD**
   ```yaml
   # GitHub Actions example
   - name: Deploy
     env:
       PRIVATE_KEY: ${{ secrets.EIGENX_PRIVATE_KEY }}
     run: eigenx app deploy
   ```

3. **Generate Dedicated Keys for EigenX**
   ```bash
   eigenx auth generate --store
   ```

4. **Check Key Source**
   ```bash
   eigenx auth whoami
   # Shows where key is loaded from
   ```

### DON'Ts

1. **Never Share Private Keys**
   - Don't commit to Git
   - Don't share in messages
   - Don't log or print them

2. **Avoid Command Flags for Keys**
   ```bash
   # Bad - saved in shell history
   eigenx app deploy --private-key 0xabc...

   # Good - use keyring or env var
   eigenx app deploy
   ```

3. **Don't Reuse Keys Across Services**
   - Use unique keys for EigenX
   - Separate from other dApps

## Keyring Storage Locations

The OS keyring stores keys encrypted:

- **macOS:** Keychain Access
- **Linux:** Secret Service (GNOME Keyring, KWallet)
- **Windows:** Windows Credential Manager

### Viewing in OS

#### macOS
1. Open Keychain Access
2. Search for "eigenx"
3. View item details (requires password)

#### Linux (GNOME)
```bash
secret-tool search application eigenx
```

#### Windows
1. Open Credential Manager
2. Navigate to "Windows Credentials"
3. Look for "eigenx" entries

## Troubleshooting

### "No authentication found"

1. Check if key is stored:
   ```bash
   eigenx auth whoami
   ```

2. If not, login:
   ```bash
   eigenx auth login
   ```

### "Keyring error: The specified item could not be found"

The keyring doesn't have a key for this environment:

```bash
# Check current environment
eigenx environment show

# Login for this environment
eigenx auth login
```

### "Insufficient funds"

Your wallet needs ETH for gas:

1. Check your address:
   ```bash
   eigenx auth whoami
   ```

2. Get testnet ETH:
   - [Google Cloud Sepolia Faucet](https://cloud.google.com/application/web3/faucet/ethereum/sepolia)
   - [Alchemy Sepolia Faucet](https://sepoliafaucet.com/)

### Permission Denied (Linux)

If keyring access fails on Linux:

```bash
# Install required packages
sudo apt-get install gnome-keyring libsecret-1-0

# For headless systems
sudo apt-get install libsecret-tools
```

### CI/CD Authentication

For GitHub Actions:

```yaml
name: Deploy
on: push

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Install EigenX
        run: |
          curl -fsSL https://eigenx-scripts.s3.us-east-1.amazonaws.com/install-eigenx.sh | bash

      - name: Deploy
        env:
          PRIVATE_KEY: ${{ secrets.EIGENX_PRIVATE_KEY }}
        run: |
          eigenx app deploy
```

## Key Rotation

To rotate your key:

1. Generate new key:
   ```bash
   eigenx auth generate --store
   # Save the new address
   ```

2. Transfer app ownership (if needed):
   ```bash
   # Use old key to transfer ownership to new address
   # (Feature coming soon)
   ```

3. Remove old key:
   ```bash
   eigenx auth logout
   ```

## Account Requirements

### Allowlisting

New accounts need to be allowlisted to deploy apps:

1. Generate or use existing key:
   ```bash
   eigenx auth generate --store
   # or
   eigenx auth login
   ```

2. Get your address:
   ```bash
   eigenx auth whoami
   ```

3. Submit onboarding request:
   - Visit [onboarding.eigencloud.xyz](https://onboarding.eigencloud.xyz)
   - Provide your address
   - Wait for approval

### Checking Allowlist Status

```bash
# Deploy will show if you're not allowlisted
eigenx app deploy

# Error: Address 0x... is not allowlisted
```

## Advanced Usage

### Multiple Profiles

Use different keys for different projects:

```bash
# Personal projects
PRIVATE_KEY=$PERSONAL_KEY eigenx app deploy

# Work projects
PRIVATE_KEY=$WORK_KEY eigenx app deploy
```

### Hardware Wallets

Hardware wallet support is not yet available but planned for future releases.

### Key Derivation

To use HD wallets (hierarchical deterministic):

```javascript
// In your local script (not in TEE)
const { HDNodeWallet } = require('ethers');
const wallet = HDNodeWallet.fromMnemonic(mnemonic);
const child = wallet.derivePath("m/44'/60'/0'/0/0");
console.log(child.privateKey);
```

## Related Documentation

- [Core Concepts - Keys](CONCEPTS.md#overview-of-keys)
- [Security Best Practices](CONCEPTS.md#security-best-practices)
- [Quick Start](../README.md#quick-start)
