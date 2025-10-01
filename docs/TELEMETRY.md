# Telemetry & Privacy

This document explains what telemetry data the EigenX CLI collects, why we collect it, and how to control it.

## Quick Controls

```bash
# Check current status
eigenx telemetry --status

# Disable telemetry
eigenx telemetry --disable

# Re-enable telemetry
eigenx telemetry --enable
```

## Why We Collect Telemetry

Telemetry helps us:
- Identify and fix common errors
- Understand which features are used most
- Improve performance and user experience
- Prioritize development efforts

## What We Collect

### Command Usage
- Commands executed (e.g., `app deploy`, `auth login`)
- Command success/failure status
- Execution time
- Error types (not error messages with sensitive data)

### System Information
- Operating system and version
- CLI version
- System architecture (x64, ARM64)
- CI environment detection

### Geographic Information
- Country and city (IP-based)
- No street-level location data

### Anonymous Identifiers
- Random UUID per installation
- No personal information
- No wallet addresses
- No private keys

### Deployment Context
- Deployment environment (sepolia, mainnet-alpha)
- Template language used (typescript, python, etc.)
- Feature flags enabled

## What We DON'T Collect

### Never Collected
- ❌ Private keys or mnemonics
- ❌ Wallet addresses
- ❌ Environment variables or secrets
- ❌ Application source code
- ❌ File paths or project names
- ❌ Docker image contents
- ❌ API keys or credentials
- ❌ Personal information
- ❌ Blockchain transaction details

### Explicitly Excluded
- Command arguments that might contain sensitive data
- Error messages that might include paths or secrets
- Network request/response bodies
- User input to interactive prompts

## Data Storage & Processing

### Infrastructure
- Data sent to PostHog (analytics platform)
- Stored in US data centers (https://us.i.posthog.com)
- Encrypted in transit (HTTPS)

### Access Control
- Limited to EigenX development team
- Used only for product improvement
- Never shared with third parties
- Never used for marketing

## Telemetry Configuration

### Global Settings

Telemetry preferences are stored globally and persist across all projects:

```yaml
# ~/.config/eigenx/config.yaml
TelemetryEnabled: true  # or false
UserUUID: "random-uuid-here"
```

### Environment Variables

Override telemetry settings temporarily:

```bash
# Disable for single command
EIGENX_TELEMETRY_DISABLED=true eigenx app deploy

# Disable for session
export EIGENX_TELEMETRY_DISABLED=true
```

### CI/CD Environments

Telemetry is automatically disabled in CI environments when the `CI` environment variable is set to `true`.

To force enable in CI:
```bash
EIGENX_TELEMETRY_FORCE_ENABLE=true eigenx app deploy
```

## Telemetry Events

### Events We Track

| Event | Data Collected |
|-------|----------------|
| `cli_started` | Command, version, OS |
| `command_executed` | Command name, duration, success |
| `error_occurred` | Error type, command context |
| `auth_method_used` | Method type (keyring/env/flag) |
| `deployment_initiated` | Environment, template type |
| `app_created` | Template language |
| `feature_used` | Feature name (e.g., TLS, watch mode) |

### Example Event

```json
{
  "event": "command_executed",
  "properties": {
    "command": "app deploy",
    "duration_ms": 45000,
    "success": true,
    "environment": "sepolia",
    "cli_version": "1.0.0",
    "os": "darwin",
    "arch": "arm64"
  }
}
```

## First Run Experience

On first run, the CLI will:
1. Generate a random UUID for anonymous identification
2. Enable telemetry by default
3. Show a one-time notice about telemetry
4. Provide instructions to disable if desired

```
Welcome to EigenX CLI!

We collect anonymous usage data to improve the CLI.
To opt out, run: eigenx telemetry --disable
Learn more: https://github.com/Layr-Labs/eigenx-cli/blob/main/docs/TELEMETRY.md
```

## Compliance & Privacy

### GDPR Compliance
- No personal data collected
- Right to deletion (via disabling telemetry)
- Data minimization principle followed
- Purpose limitation respected

### Data Subject Rights
While we don't collect personal data, you can:
- Disable collection anytime
- No historical data retained after opt-out
- Request data deletion by contacting support

## Debugging Telemetry

### View What Would Be Sent

Debug mode (coming soon):
```bash
EIGENX_TELEMETRY_DEBUG=true eigenx app deploy
# Shows telemetry events without sending
```

### Check Configuration

```bash
# View current settings
eigenx telemetry --status

# Check config file
cat ~/.config/eigenx/config.yaml
```

## Corporate Environments

### Firewall Configuration

Telemetry sends data to:
- `app.posthog.com` (HTTPS, port 443)

### Proxy Support

Telemetry respects standard proxy environment variables:
```bash
export HTTPS_PROXY=http://proxy.company.com:8080
eigenx app deploy
```

### Disable for All Users

System administrators can disable telemetry organization-wide:

```bash
# Create system-wide config
echo "TelemetryEnabled: false" > /etc/eigenx/config.yaml
```

## Frequently Asked Questions

### Can I see the telemetry code?

Yes! The telemetry implementation is open source:
- [pkg/telemetry/](https://github.com/Layr-Labs/eigenx-cli/tree/main/pkg/telemetry)

### Does telemetry affect performance?

No. Telemetry is:
- Asynchronous (non-blocking)
- Batched (efficient)
- Minimal overhead (<5ms per event)
- Fails silently if issues occur

### What if I'm behind a firewall?

If telemetry cannot reach PostHog:
- Commands continue normally
- No errors shown to user
- No retry attempts
- No data cached locally

### Can I enable telemetry for specific commands?

```bash
# Disable globally
eigenx telemetry --disable

# Enable for single command
EIGENX_TELEMETRY_FORCE_ENABLE=true eigenx app deploy
```

### Is telemetry data anonymized?

Yes. We use:
- Random UUIDs (not linked to identity)
- No PII collection
- IP addresses not stored
- Aggregated statistics only

### How do I know telemetry is disabled?

```bash
eigenx telemetry --status

# Output when disabled:
# Telemetry is currently: DISABLED
# Anonymous usage data collection is turned off
```

## Changes to Telemetry

We may update what we collect. Changes will be:
- Documented in release notes
- Updated in this document
- Never retroactive
- Always respect your opt-out choice

## Contact

Questions or concerns about telemetry?
- Open an issue: [GitHub Issues](https://github.com/Layr-Labs/eigenx-cli/issues)
- Email: privacy@eigenlabs.org

## Related Documentation

- [Installation Guide](INSTALLATION.md)
- [Configuration](COMMANDS.md#configuration-commands)
- [Development](DEVELOPMENT.md)