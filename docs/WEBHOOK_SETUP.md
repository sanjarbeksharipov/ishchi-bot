# Webhook and Long Polling Setup Guide

This bot supports both **Long Polling** (for local development) and **Webhook** (for production deployment).

## Long Polling Mode (Local Development)

Long polling is the default mode and is ideal for local development.

### Configuration

In your `.env` file:

```bash
BOT_MODE=polling
BOT_POLLER=10s  # Polling timeout
```

### How it works

- The bot actively requests updates from Telegram servers
- No public URL required
- Works behind firewalls and NAT
- Easier to debug locally

### Usage

Simply run the bot:

```bash
make run
# or
go run cmd/main.go
```

---

## Webhook Mode (Production)

Webhook mode is recommended for production deployments as it's more efficient and scalable.

### Prerequisites

1. **Public HTTPS URL**: You need a publicly accessible domain with HTTPS
2. **Valid SSL Certificate**: Telegram requires HTTPS for webhooks
3. **Open Port**: The webhook port (default 8443) must be accessible from the internet

### Configuration

In your `.env` file:

```bash
BOT_MODE=webhook
BOT_WEBHOOK_URL=https://yourdomain.com/webhook
BOT_WEBHOOK_LISTEN=:8443
BOT_WEBHOOK_PORT=8443
```

### Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `BOT_MODE` | Bot operation mode | `webhook` or `polling` |
| `BOT_WEBHOOK_URL` | Public HTTPS URL for webhook | `https://example.com/webhook` |
| `BOT_WEBHOOK_LISTEN` | Local address to listen on | `:8443` or `0.0.0.0:8443` |
| `BOT_WEBHOOK_PORT` | Port for webhook server | `8443`, `443`, or `8080` |

### How it works

- Telegram sends updates directly to your server
- More efficient (no constant polling)
- Lower latency
- Better for high-traffic bots

### Deployment Example with Docker

1. Update your `.env` file:

```bash
BOT_MODE=webhook
BOT_WEBHOOK_URL=https://yourdomain.com/webhook
BOT_WEBHOOK_LISTEN=:8443
BOT_WEBHOOK_PORT=8443
```

2. Ensure your `docker-compose.yml` exposes the webhook port:

```yaml
services:
  bot:
    ports:
      - "8443:8443"
```

3. Deploy with reverse proxy (nginx example):

```nginx
server {
    listen 443 ssl;
    server_name yourdomain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location /webhook {
        proxy_pass http://localhost:8443;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Allowed Ports

Telegram only accepts webhooks on these ports:
- 443 (HTTPS default)
- 80 (HTTP - not recommended)
- 88
- 8443 (commonly used)

### Testing Webhook Locally

For local webhook testing, you can use tools like:

- **ngrok**: `ngrok http 8443`
- **localtunnel**: `lt --port 8443`

Then set `BOT_WEBHOOK_URL` to the provided HTTPS URL.

---

## Switching Between Modes

Simply change the `BOT_MODE` environment variable and restart the bot:

```bash
# For local development
BOT_MODE=polling

# For production
BOT_MODE=webhook
```

The bot will automatically configure itself based on the mode.

---

## Troubleshooting

### Webhook Issues

1. **"Webhook failed" error**
   - Ensure your URL is publicly accessible
   - Verify SSL certificate is valid
   - Check that the port is open in your firewall

2. **No updates received**
   - Verify `BOT_WEBHOOK_URL` is correct
   - Check bot logs for errors
   - Ensure Telegram can reach your server

3. **SSL/TLS errors**
   - Use a valid SSL certificate (Let's Encrypt recommended)
   - Telegram doesn't accept self-signed certificates in production

### Polling Issues

1. **Slow updates**
   - Increase `BOT_POLLER` timeout (e.g., `30s`)
   - Check network connectivity

2. **Connection timeouts**
   - Verify bot token is correct
   - Check internet connection
   - Ensure Telegram API is accessible

---

## Best Practices

1. **Use polling for development**: Easier to debug and doesn't require public URL
2. **Use webhook for production**: More efficient and scalable
3. **Monitor logs**: Both modes log their configuration on startup
4. **Secure your webhook**: Use HTTPS and consider IP whitelisting for Telegram's IPs
5. **Handle graceful shutdown**: The bot properly stops both polling and webhook on shutdown

---

## Security Considerations

### Webhook Mode
- Always use HTTPS (required by Telegram)
- Consider implementing webhook secret validation
- Restrict access to webhook endpoint
- Use environment variables for sensitive data

### Polling Mode
- Keep bot token secure
- Don't expose polling bot to public internet
- Use for development only

---

## Additional Resources

- [Telegram Bot API - Webhooks](https://core.telegram.org/bots/api#setwebhook)
- [Telegram Bot API - Getting Updates](https://core.telegram.org/bots/api#getting-updates)
- [Telebot Documentation](https://github.com/tucnak/telebot)
