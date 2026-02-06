# Telegram Bot - Ishchi Bot

A production-ready Telegram bot built with Go, featuring PostgreSQL storage, payment processing, and support for both webhook and long polling modes.

## Features

- ✅ **Dual Mode Support**: Long polling for development, webhook for production
- ✅ **PostgreSQL Database**: Robust data persistence
- ✅ **Payment Processing**: Integrated payment handling
- ✅ **Admin Panel**: Comprehensive admin controls
- ✅ **Job Management**: Post and manage job listings
- ✅ **Graceful Shutdown**: Proper cleanup on exit
- ✅ **Docker Support**: Easy deployment with Docker Compose

## Quick Start

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 14 or higher
- Telegram Bot Token (from [@BotFather](https://t.me/botfather))

### Installation

1. **Clone the repository**

```bash
git clone <repository-url>
cd ishchi-bot
```

2. **Set up environment variables**

```bash
cp .env.example .env
# Edit .env with your configuration
```

3. **Run database migrations**

```bash
make migrate-up
```

4. **Run the bot**

```bash
# For local development (long polling)
make run

# Or with Docker
docker-compose up
```

## Configuration

### Bot Modes

The bot supports two modes of operation:

#### 🔹 Long Polling (Local Development)

Best for local development and testing.

```bash
BOT_MODE=polling
BOT_POLLER=10s
```

#### 🔹 Webhook (Production)

Recommended for production deployments.

```bash
BOT_MODE=webhook
BOT_WEBHOOK_URL=https://yourdomain.com/webhook
BOT_WEBHOOK_LISTEN=:8443
BOT_WEBHOOK_PORT=8443
```

**📖 For detailed webhook setup instructions, see [docs/WEBHOOK_SETUP.md](docs/WEBHOOK_SETUP.md)**

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `BOT_TOKEN` | Telegram bot token | - | ✅ |
| `BOT_MODE` | Operation mode (`polling` or `webhook`) | `polling` | ❌ |
| `BOT_WEBHOOK_URL` | Public webhook URL | - | ✅ (webhook mode) |
| `BOT_WEBHOOK_LISTEN` | Webhook listen address | `:8443` | ❌ |
| `BOT_WEBHOOK_PORT` | Webhook port | `8443` | ❌ |
| `BOT_POLLER` | Polling timeout | `10s` | ❌ |
| `BOT_CHANNEL_ID` | Channel ID for posts | `0` | ❌ |
| `BOT_ADMIN_IDS` | Comma-separated admin IDs | - | ✅ |
| `BOT_ADMIN_GROUP_ID` | Admin group ID | `0` | ❌ |
| `BOT_USERNAME` | Bot username | - | ✅ |
| `DB_HOST` | Database host | `localhost` | ✅ |
| `DB_PORT` | Database port | `5432` | ✅ |
| `DB_USER` | Database user | `postgres` | ✅ |
| `DB_PASSWORD` | Database password | - | ✅ |
| `DB_NAME` | Database name | `telegram_bot` | ✅ |
| `DB_MAX_CONNECTIONS` | Max DB connections | `25` | ❌ |
| `APP_ENV` | Environment (`development`/`production`) | `development` | ❌ |
| `LOG_LEVEL` | Log level | `info` | ❌ |
| `CARD_NUMBER` | Payment card number | - | ✅ |
| `CARD_HOLDER_NAME` | Card holder name | - | ✅ |

## Project Structure

```
.
├── bot/                    # Bot core logic
│   ├── handlers/          # Message and callback handlers
│   ├── models/            # Data models
│   └── routes.go          # Route registration
├── cmd/                   # Application entry point
│   └── main.go
├── config/                # Configuration management
├── docs/                  # Documentation
│   └── WEBHOOK_SETUP.md  # Webhook setup guide
├── migrations/            # Database migrations
├── pkg/                   # Shared packages
│   ├── keyboards/        # Telegram keyboards
│   ├── logger/           # Logging utilities
│   └── messages/         # Message templates
├── service/              # Business logic services
├── storage/              # Data access layer
│   └── postgres/         # PostgreSQL implementation
├── docker-compose.yml    # Docker Compose configuration
├── Dockerfile            # Docker image definition
├── Makefile             # Build and run commands
└── .env.example         # Environment variables template
```

## Available Commands

```bash
# Development
make run                 # Run the bot locally
make build              # Build the binary
make test               # Run tests

# Database
make migrate-up         # Apply migrations
make migrate-down       # Rollback migrations
make migrate-create     # Create new migration

# Docker
make docker-build       # Build Docker image
make docker-up          # Start with Docker Compose
make docker-down        # Stop Docker containers
make docker-logs        # View Docker logs

# Utilities
make clean              # Clean build artifacts
make lint               # Run linter
```

## Development

### Local Development Setup

1. **Start PostgreSQL**

```bash
docker-compose up -d postgres
```

2. **Run migrations**

```bash
make migrate-up
```

3. **Start the bot in polling mode**

```bash
# Ensure BOT_MODE=polling in .env
make run
```

### Production Deployment

1. **Set up your server with HTTPS**

2. **Configure webhook mode**

```bash
BOT_MODE=webhook
BOT_WEBHOOK_URL=https://yourdomain.com/webhook
```

3. **Deploy with Docker**

```bash
docker-compose up -d
```

For detailed webhook setup, see [docs/WEBHOOK_SETUP.md](docs/WEBHOOK_SETUP.md)

## Architecture

This bot follows a clean architecture pattern:

- **Handlers**: Process incoming updates and callbacks
- **Services**: Business logic and orchestration
- **Storage**: Data persistence layer
- **Models**: Data structures and domain entities

For more details, see [ARCHITECTURE.md](ARCHITECTURE.md)

## Features in Detail

### Job Management
- Post job listings
- Edit job details
- Delete jobs
- Automatic expiry handling

### Payment System
- Payment request handling
- Admin approval workflow
- Payment confirmation

### Admin Panel
- User management
- Job moderation
- Payment approvals
- Statistics and reporting

## Troubleshooting

### Bot not receiving updates

**Polling mode:**
- Check bot token is correct
- Verify internet connection
- Ensure no other instance is running

**Webhook mode:**
- Verify webhook URL is publicly accessible
- Check SSL certificate is valid
- Ensure port is open in firewall
- Review bot logs for errors

### Database connection issues

- Verify PostgreSQL is running
- Check database credentials in `.env`
- Ensure database exists and migrations are applied

### Common Errors

| Error | Solution |
|-------|----------|
| "BOT_TOKEN environment variable is required" | Set `BOT_TOKEN` in `.env` |
| "BOT_WEBHOOK_URL is required when BOT_MODE=webhook" | Set `BOT_WEBHOOK_URL` or switch to polling mode |
| "Failed to initialize storage" | Check database connection and credentials |

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linter
5. Submit a pull request

## License

[Your License Here]

## Support

For issues and questions:
- Create an issue in the repository
- Check existing documentation in `/docs`
- Review [WEBHOOK_SETUP.md](docs/WEBHOOK_SETUP.md) for webhook-specific help

## Acknowledgments

Built with:
- [telebot](https://github.com/tucnak/telebot) - Telegram Bot framework
- [PostgreSQL](https://www.postgresql.org/) - Database
- [Go](https://golang.org/) - Programming language
