# Installation Guide

This guide covers all installation methods for the EigenX CLI, including quick install scripts, manual installation, and building from source.

## Quick Installation

### macOS/Linux
```bash
curl -fsSL https://eigenx-scripts.s3.us-east-1.amazonaws.com/install-eigenx.sh | bash
```

### Windows
```bash
curl -fsSL https://eigenx-scripts.s3.us-east-1.amazonaws.com/install-eigenx.ps1 | powershell -
```

## Installation Options

### Development Version

To install the development version (installed as `eigenx-dev`):

```bash
# macOS/Linux
curl -fsSL https://eigenx-scripts.s3.us-east-1.amazonaws.com/install-eigenx.sh | bash -s -- --dev

# Windows
curl -fsSL https://eigenx-scripts.s3.us-east-1.amazonaws.com/install-eigenx.ps1 | powershell - -Dev
```

**Note:** Both `eigenx` and `eigenx-dev` can coexist on the same system.

### Custom Installation Directory

By default, the CLI installs to:
- macOS/Linux: `~/bin/eigenx`
- Windows: `%USERPROFILE%\bin\eigenx.exe`

To install to a custom directory:

```bash
# macOS/Linux
curl -fsSL https://eigenx-scripts.s3.us-east-1.amazonaws.com/install-eigenx.sh | bash -s -- --dir /custom/path

# Windows
curl -fsSL https://eigenx-scripts.s3.us-east-1.amazonaws.com/install-eigenx.ps1 | powershell - -Dir C:\custom\path
```

## Manual Installation

If you prefer to install manually:

1. Download the latest release from [GitHub Releases](https://github.com/Layr-Labs/eigenx-cli/releases)
2. Choose the appropriate binary for your platform:
   - `eigenx-darwin-amd64` - macOS (Intel)
   - `eigenx-darwin-arm64` - macOS (Apple Silicon)
   - `eigenx-linux-amd64` - Linux (x64)
   - `eigenx-linux-arm64` - Linux (ARM64)
   - `eigenx-windows-amd64.exe` - Windows (x64)
3. Rename the binary to `eigenx` (or `eigenx.exe` on Windows)
4. Make it executable (macOS/Linux): `chmod +x eigenx`
5. Move it to a directory in your PATH

## Building from Source

### Prerequisites

- Go 1.23.6 or higher
- Make
- Git

### Build Steps

```bash
# Clone the repository
git clone https://github.com/Layr-Labs/eigenx-cli
cd eigenx-cli

# Build for development (default)
make build

# Build for production
GO_TAGS=prod make build

# Install to ~/bin/
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

The binary will be created at `./bin/eigenx`.


## Verifying Installation

After installation, verify the CLI is working:

```bash
# Check version
eigenx version

# View help
eigenx --help

# Check authentication status
eigenx auth whoami
```

## Upgrading

To upgrade to the latest version:

```bash
eigenx upgrade
```

Or reinstall using the installation script.

## Uninstallation

### macOS/Linux
```bash
# Remove binary
rm ~/bin/eigenx

# Remove configuration (optional)
rm -rf ~/.config/eigenx

# Remove stored credentials (optional)
eigenx auth logout  # Run before removing binary
```

### Windows
```powershell
# Remove binary
Remove-Item "$env:USERPROFILE\bin\eigenx.exe"

# Remove configuration (optional)
Remove-Item -Recurse "$env:APPDATA\eigenx"
```

## Troubleshooting

### Command Not Found

If `eigenx` is not found after installation:

1. Check if the binary exists:
   ```bash
   ls ~/bin/eigenx  # macOS/Linux
   dir %USERPROFILE%\bin\eigenx.exe  # Windows
   ```

2. Add the installation directory to your PATH:
   ```bash
   # macOS/Linux (add to ~/.bashrc or ~/.zshrc)
   export PATH="$HOME/bin:$PATH"

   # Windows (PowerShell)
   $env:Path += ";$env:USERPROFILE\bin"
   ```

3. Reload your shell configuration:
   ```bash
   source ~/.bashrc  # or ~/.zshrc
   ```

### Permission Denied

On macOS/Linux, make the binary executable:
```bash
chmod +x ~/bin/eigenx
```

### macOS Security Warning

On macOS, you may see "eigenx cannot be opened because the developer cannot be verified."

To resolve:
1. Go to System Preferences → Security & Privacy
2. Click "Allow Anyway" for eigenx
3. Or run: `xattr -d com.apple.quarantine ~/bin/eigenx`

## Next Steps

After installation:
1. [Configure authentication](AUTHENTICATION.md)
2. Check the [Quick Start](../README.md#quick-start) guide
3. View available [commands](COMMANDS.md)