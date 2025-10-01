# TLS/HTTPS Setup Guide

This guide walks you through enabling HTTPS for your EigenX application with automatic SSL certificate management.

## Overview

EigenX uses Caddy server as a reverse proxy to:
- Automatically obtain and renew SSL certificates from Let's Encrypt
- Handle HTTPS termination inside the TEE
- Route traffic to your application

## Quick Setup

### Step 1: Add TLS Configuration

From your project directory:

```bash
eigenx app configure tls
```

This creates two files:
- `Caddyfile` - Caddy server configuration
- `.env.example.tls` - TLS environment variable template

### Step 2: Configure Environment Variables

Add the TLS variables to your `.env` file:

```bash
# Required
DOMAIN=yourdomain.com    # Your domain name
APP_PORT=3000            # Your app's internal port

# Optional (recommended for initial setup)
ENABLE_CADDY_LOGS=true   # Enable Caddy debug logs
ACME_STAGING=true        # Use staging certificates (for testing)
```

### Step 3: Set Up DNS

Create an A record pointing your domain to the instance IP:

1. Get your instance IP:
   ```bash
   eigenx app info
   # Look for: Instance IP: xxx.xxx.xxx.xxx
   ```

2. Add DNS A record:
   - **Type:** A
   - **Name:** @ (or subdomain)
   - **Value:** `<instance-ip>`
   - **TTL:** 300 (5 minutes)

3. Verify DNS propagation:
   ```bash
   dig yourdomain.com
   nslookup yourdomain.com
   ```

### Step 4: Deploy

Deploy or upgrade your application:

```bash
eigenx app upgrade
```

### Step 5: Test

Once deployed, test your HTTPS connection:

```bash
# Check HTTPS
curl https://yourdomain.com

# View certificate details
openssl s_client -connect yourdomain.com:443 -servername yourdomain.com
```

## Configuration Options

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `DOMAIN` | Your domain name | `api.example.com` |
| `APP_PORT` | Port your app listens on internally | `3000` |

### Optional Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ENABLE_CADDY_LOGS` | `false` | Enable Caddy debug logs |
| `ACME_STAGING` | `false` | Use Let's Encrypt staging environment |
| `ACME_FORCE_ISSUE` | `false` | Force certificate reissue |
| `CLIENT_MAX_BODY_SIZE` | `100m` | Maximum request body size |

## Staging vs Production Certificates

### Staging Certificates (Testing)

Use staging certificates to test your setup without hitting rate limits:

```bash
ACME_STAGING=true
```

**Characteristics:**
- Unlimited requests
- Browser shows security warning (untrusted CA)
- Perfect for testing configuration

### Production Certificates

For production use:

```bash
ACME_STAGING=false
```

**Rate Limits:**
- 5 certificates per week per domain
- 5 duplicate certificates per week
- 300 new orders per 3 hours

### Switching from Staging to Production

When ready for production certificates:

1. Update `.env`:
   ```bash
   ACME_STAGING=false
   ACME_FORCE_ISSUE=true  # Forces new certificate
   ```

2. Deploy:
   ```bash
   eigenx app upgrade
   ```

3. After successful deployment, remove force flag:
   ```bash
   ACME_FORCE_ISSUE=false
   ```

## Multiple Domains

To support multiple domains or subdomains:

```bash
DOMAIN=example.com,www.example.com,api.example.com
```

Each domain needs its own DNS A record pointing to the instance IP.

## Troubleshooting

### Certificate Not Issuing

1. **Check DNS propagation:**
   ```bash
   dig yourdomain.com
   # Should return your instance IP
   ```

2. **Enable debug logs:**
   ```bash
   ENABLE_CADDY_LOGS=true
   ```
   Then check logs:
   ```bash
   eigenx app logs
   ```

3. **Verify ports are accessible:**
   - Port 80 must be accessible for ACME challenge
   - Port 443 for HTTPS traffic

### "Too Many Certificates" Error

You've hit Let's Encrypt rate limits. Solutions:
1. Wait 1 week for limit reset
2. Use staging certificates for testing
3. Use a subdomain instead

### Certificate Shows as Invalid

If using staging certificates, this is expected. Browsers will show a warning because staging certificates are from an untrusted CA.

To fix:
1. Switch to production certificates (see above)
2. Or accept the security warning in your browser (for testing only)

### Connection Refused

1. **Check app is running:**
   ```bash
   eigenx app info
   # Should show: Status: Running
   ```

2. **Verify APP_PORT matches your application:**
   ```bash
   # In your app
   app.listen(3000)  # Must match APP_PORT=3000
   ```

3. **Check Caddy logs:**
   ```bash
   eigenx app logs | grep caddy
   ```

## Advanced Configuration

### Custom Caddyfile

The default Caddyfile handles most use cases. To customize:

1. Edit `Caddyfile` in your project
2. Common customizations:

```caddyfile
{$DOMAIN} {
    # Custom headers
    header {
        X-Custom-Header "value"
        -Server  # Remove server header
    }

    # Rate limiting
    rate_limit {
        zone dynamic {
            key {remote_host}
            events 100
            window 1m
        }
    }

    # Custom error pages
    handle_errors {
        respond "Custom error page"
    }

    reverse_proxy 127.0.0.1:{$APP_PORT}
}
```

### Websocket Support

Websockets are supported by default. The Caddyfile includes:

```caddyfile
header {
    Upgrade {http.request.header.Upgrade}
    Connection {http.request.header.Connection}
}
```

### Health Checks

Add a health check endpoint:

```caddyfile
handle /health {
    respond "OK" 200
}
```

## Security Considerations

1. **Private Key Security:** SSL private keys are generated and stored inside the TEE, never exposed
2. **Certificate Renewal:** Certificates auto-renew 30 days before expiration
3. **HSTS:** Enabled by default with 1-year max-age
4. **TLS Versions:** Only TLS 1.2 and 1.3 are supported

## Monitoring

### Certificate Expiration

Check certificate expiration:

```bash
echo | openssl s_client -connect yourdomain.com:443 2>/dev/null | \
  openssl x509 -noout -dates
```

### Caddy Metrics

View Caddy metrics in logs:

```bash
eigenx app logs | grep -E "certificate|tls|acme"
```

## Best Practices

1. **Test with staging first** - Avoid rate limits during setup
2. **Use specific subdomains** - Easier to manage than wildcards
3. **Monitor certificate expiration** - Set up alerts for expiration
4. **Keep Caddy logs enabled initially** - Disable after stable
5. **Use DNS with low TTL** - Allows quick changes during setup

## Migration from HTTP

If migrating from HTTP to HTTPS:

1. Keep HTTP app running
2. Set up new HTTPS deployment
3. Test HTTPS thoroughly
4. Update DNS to new instance
5. Add redirects from old URLs if needed
6. Terminate old HTTP deployment

## Next Steps

- Review [security best practices](CONCEPTS.md#security-best-practices)
- Set up monitoring for your domain
- Configure CDN if needed (CloudFlare, etc.)