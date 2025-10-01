# Command Reference

Complete reference for all EigenX CLI commands.

## Global Options

These options work with any command:

- `--env` or `-e` - Set deployment environment (default: sepolia)
- `--private-key` - Provide private key directly (not recommended)
- `--help` or `-h` - Show help for any command
- `--version` or `-v` - Show CLI version

## Authentication Commands

### `eigenx auth generate`

Generate a new private key.

**Aliases:** `gen`, `new`

**Options:**
- `--store` - Store the generated key in OS keyring

**Example:**
```bash
eigenx auth generate --store
```

### `eigenx auth login`

Store an existing private key in the OS keyring.

**Interactive:** Prompts for private key if not provided.

**Example:**
```bash
eigenx auth login
```

### `eigenx auth whoami`

Display current authentication status and wallet address.

**Example:**
```bash
eigenx auth whoami
```

### `eigenx auth list`

List all stored private keys by environment.

**Example:**
```bash
eigenx auth list
```

### `eigenx auth logout`

Remove private key from OS keyring.

**Options:**
- `--all` - Remove keys for all environments

**Example:**
```bash
eigenx auth logout
eigenx auth logout --all
```

## Project Management Commands

### `eigenx app create [name] [language]`

Create a new project from a template.

**Parameters:**
- `name` - Project name (optional, interactive if not provided)
- `language` - Template language: `typescript`, `python`, `golang`, `rust` (optional, interactive if not provided)

**Options:**
- `--path` - Directory to create project in (default: current directory)

**Example:**
```bash
eigenx app create my-app typescript
eigenx app create  # Interactive mode
```

### `eigenx app configure tls`

Add TLS/HTTPS configuration files to your project.

**Creates:**
- `Caddyfile` - Caddy server configuration
- `.env.example.tls` - TLS environment variables template

**Example:**
```bash
eigenx app configure tls
```

### `eigenx app name <app-id|name> [new-name]`

Set, change, or remove a friendly name for your app.

**Parameters:**
- `app-id|name` - App ID (0x...) or existing friendly name
- `new-name` - New friendly name (optional, removes name if not provided)

**Example:**
```bash
eigenx app name 0x1234... my-trading-bot  # Set name
eigenx app name my-trading-bot new-name   # Change name
eigenx app name new-name                  # Remove name
```

## Deployment Commands

### `eigenx app deploy [image_ref]`

Deploy a new application to TEE.

**Parameters:**
- `image_ref` - Docker image reference (optional, builds and pushes if not provided)

**Options:**
- `--dockerfile` - Path to Dockerfile (default: ./Dockerfile)
- `--env-file` - Path to environment file (default: ./.env)
- `--name` - App name for context detection

**Process:**
1. Build Docker image (unless image_ref provided)
2. Push to registry
3. Encrypt environment variables
4. Deploy to TEE via smart contract

**Example:**
```bash
eigenx app deploy                           # Build, push, and deploy
eigenx app deploy myregistry/myapp:v1.0    # Deploy existing image
```

### `eigenx app upgrade <app-id|name> [image_ref]`

Upgrade an existing application.

**Parameters:**
- `app-id|name` - App ID (0x...) or friendly name (optional if in project directory)
- `image_ref` - Docker image reference (optional, builds and pushes if not provided)

**Options:**
- Same as `deploy`

**Example:**
```bash
eigenx app upgrade my-app
eigenx app upgrade my-app myregistry/myapp:v2.0
eigenx app upgrade  # Auto-detects from current directory
```

## Lifecycle Management Commands

### `eigenx app start [app-id|name]`

Start a stopped application (starts GCP instance).

**Parameters:**
- `app-id|name` - App ID or friendly name (optional, interactive if not provided)

**Example:**
```bash
eigenx app start my-app
eigenx app start  # Interactive selection
```

### `eigenx app stop [app-id|name]`

Stop a running application (stops GCP instance).

**Parameters:**
- `app-id|name` - App ID or friendly name (optional, interactive if not provided)

**Example:**
```bash
eigenx app stop my-app
```

### `eigenx app terminate [app-id|name]`

Permanently terminate an application (terminates GCP instance).

**Parameters:**
- `app-id|name` - App ID or friendly name (optional, interactive if not provided)

**Example:**
```bash
eigenx app terminate my-app
```

## Monitoring Commands

### `eigenx app list`

List all deployed applications.

**Example:**
```bash
eigenx app list
```

### `eigenx app info [app-id|name]`

Show detailed information about an application.

**Parameters:**
- `app-id|name` - App ID or friendly name (optional, interactive if not provided)

**Options:**
- `--watch` or `-w` - Continuously poll for updates

**Example:**
```bash
eigenx app info my-app
eigenx app info --watch  # Live updates
```

### `eigenx app logs [app-id|name]`

View application logs.

**Parameters:**
- `app-id|name` - App ID or friendly name (optional, interactive if not provided)

**Options:**
- `--watch` or `-w` - Continuously poll for new logs

**Example:**
```bash
eigenx app logs my-app
eigenx app logs --watch  # Stream logs
```

## Environment Management Commands

### `eigenx environment show`

Display the current deployment environment.

**Alias:** `env`

**Example:**
```bash
eigenx environment show
eigenx env  # Using alias
```

### `eigenx environment list`

List all available deployment environments.

**Example:**
```bash
eigenx environment list
```

### `eigenx environment set <environment>`

Change the deployment environment.

**Parameters:**
- `environment` - Environment name (e.g., sepolia, mainnet-alpha)

**Example:**
```bash
eigenx environment set mainnet-alpha
```

## Configuration Commands

### `eigenx telemetry`

Manage usage analytics settings.

**Options (exactly one required):**
- `--enable` - Enable telemetry
- `--disable` - Disable telemetry
- `--status` - Show current telemetry status

**Example:**
```bash
eigenx telemetry --status
eigenx telemetry --disable
eigenx telemetry --enable
```

### `eigenx upgrade`

Update the CLI to the latest version.

**Example:**
```bash
eigenx upgrade
```

### `eigenx version`

Display CLI version information.

**Example:**
```bash
eigenx version
```

## Interactive Features

Many commands support interactive mode when required parameters are not provided:

1. **App Selection** - Commands that need an app ID will show a list of your apps
2. **Language Selection** - `create` command shows available templates
3. **Environment Detection** - Commands auto-detect project context from current directory
4. **Confirmation Prompts** - Destructive operations ask for confirmation

## Context Detection

When run in a directory containing a Dockerfile, these commands auto-detect the project:
- `deploy` - Uses current directory as project context
- `upgrade` - Finds app name from current directory

## Authentication Priority

The CLI checks for authentication in this order:
1. `--private-key` flag
2. `PRIVATE_KEY` environment variable
3. OS keyring (set with `eigenx auth login`)

## Exit Codes

- `0` - Success
- `1` - Any error

## Getting Help

Get help for any command:

```bash
eigenx --help
eigenx app --help
eigenx app deploy --help
```