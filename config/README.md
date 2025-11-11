## Prerequisites

Before deploying, you'll need:

- **Docker** - To package and publish your application image
  - [Download Docker](https://www.docker.com/get-started/)
  - You'll also need to `docker login` to push images to your registry
- **ETH** - To pay for deployment transactions
  - For Sepolia testnet: [Google Cloud Faucet](https://cloud.google.com/application/web3/faucet/ethereum/sepolia) or [Alchemy Faucet](https://sepoliafaucet.com/)

## Deployment

```bash
# Store your private key (generate new or use existing)
eigenx auth generate --store
# OR: eigenx auth login (if you have an existing key)

eigenx app deploy username/image-name
```

The CLI will automatically detect the `Dockerfile` and build your app before deploying.

## Management & Monitoring

### App Lifecycle
```bash
eigenx app list                    # List all apps
eigenx app info [app-name]         # Get app details
eigenx app logs [app-name]         # View logs
eigenx app start [app-name]        # Start stopped app
eigenx app stop [app-name]         # Stop running app
eigenx app terminate [app-name]    # Terminate app
eigenx app upgrade [app-name] [image] # Update deployment
eigenx app configure tls            # Configure TLS
```

### App Metadata
```bash
eigenx app metadata set [app-id]  # Set app name, website, description, social links, and icon
```

## TLS Configuration (Optional)

This project includes **optional** automatic TLS certificate management using Caddy. The Caddyfile is not required - if you don't need TLS termination or prefer to handle it differently, you can simply delete the Caddyfile.

### How It Works

When a `Caddyfile` is present in your project root:
- Caddy will automatically start as a reverse proxy
- It handles TLS certificate acquisition and renewal via Let's Encrypt
- Your app runs on `APP_PORT` and Caddy forwards HTTPS traffic to it
- Certificates are stored persistently in the TEE's encrypted storage

Without a `Caddyfile`:
- Your application runs directly on the configured ports
- You can handle TLS in your application code or use an external load balancer

### Deployment Checklist

Before deploying with TLS:
1. **Configure TLS**: Run `eigenx app configure tls` to add the necessary configuration files for domain setup with private traffic termination in the TEE.
2. **DNS**: Ensure A/AAAA record points to your instance (or reserved static IP). Note: If this is your first deployment, you will need to get your IP after deployment from the `eigenx app info` command.
3. **Required configuration** in `.env`:
   ```bash
   DOMAIN=mydomain.com          # Your domain name
   APP_PORT=8000               # Your app's port
   ACME_STAGING=true           # Test with staging first to avoid rate limits
   ENABLE_CADDY_LOGS=true      # Enable logs for debugging
   ```

4. **Optional ACME configuration** (all optional, with sensible defaults):
   ```bash
   # ACME email for Let's Encrypt notifications
   ACME_EMAIL=admin@example.com

   # Certificate Authority directory URL
   # Default: https://acme-v02.api.letsencrypt.org/directory
   ACME_CA=https://acme-v02.api.letsencrypt.org/directory

   # ACME Challenge Type
   # How to prove domain ownership to Let's Encrypt
   # Both result in the same TLS certificate, just different validation methods:
   # - http-01: Uses port 80 (default)
   # - tls-alpn-01: Uses port 443
   ACME_CHALLENGE=http-01

   # Use Let's Encrypt Staging (for testing)
   # Set to true to use staging environment (certificates won't be trusted by browsers)
   # Great for testing without hitting rate limits
   ACME_STAGING=true

   # Force certificate reissue
   # Set to true to force a new certificate even if one exists
   # This will delete the existing certificate from storage and get a new one
   ACME_FORCE_ISSUE=true
   ```

5. **Customize Caddyfile** (optional):
   - Edit `Caddyfile` to match your application port
   - Modify security headers as needed
   - Configure rate limiting or other middleware

### TLS Testing & Debugging

- **Enable Caddy logs** to see TLS-related output:
  ```bash
  ENABLE_CADDY_LOGS=true
  ```

- **Use Let's Encrypt staging** for testing (avoids rate limits, but certificates won't be trusted by browsers):
  ```bash
  ACME_STAGING=true
  ```

### Local Development

For local development without TLS, leave `DOMAIN` empty or set to `localhost` in your `.env` file.

### Custom Certificates

To use custom certificates instead of Let's Encrypt, modify the `Caddyfile`:
```caddyfile
tls /path/to/cert.pem /path/to/key.pem
```

## Documentation

[EigenX CLI Documentation](https://github.com/Layr-Labs/eigenx-cli/blob/main/README.md)
