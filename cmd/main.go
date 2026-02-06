package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"telegram-bot-starter/bot"
	"telegram-bot-starter/bot/handlers"
	"telegram-bot-starter/config"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/service"
	"telegram-bot-starter/storage/postgres"

	"github.com/joho/godotenv"
	tele "gopkg.in/telebot.v4"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Failed to load .env file: %v\n", err)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	// Initialize logger
	log := logger.NewLogger("telegram-bot", cfg.App.LogLevel)
	defer func() {
		_ = logger.Cleanup(log)
	}()
	log.Info("Starting Telegram Bot...")

	// Initialize storage layer
	ctx := context.Background()
	store, err := postgres.NewPostgres(ctx, cfg, log)
	if err != nil {
		log.Fatal("Failed to initialize storage: " + err.Error())
	}
	defer store.CloseDB()
	log.Info("Storage layer initialized")

	// Create bot instance with appropriate poller based on mode
	var botSettings tele.Settings

	if cfg.Bot.Mode == "webhook" {
		// Webhook mode for production
		log.Info("Starting bot in WEBHOOK mode")

		if cfg.Bot.WebhookURL == "" {
			log.Fatal("BOT_WEBHOOK_URL is required when BOT_MODE=webhook")
		}

		botSettings = tele.Settings{
			Token: cfg.Bot.Token,
			Poller: &tele.Webhook{
				Listen:   cfg.Bot.WebhookListen,
				Endpoint: &tele.WebhookEndpoint{PublicURL: cfg.Bot.WebhookURL},
			},
		}
		log.Info(fmt.Sprintf("Webhook configured: %s (listening on %s)", cfg.Bot.WebhookURL, cfg.Bot.WebhookListen))
	} else {
		// Long polling mode for local development
		log.Info("Starting bot in LONG POLLING mode")
		botSettings = tele.Settings{
			Token:  cfg.Bot.Token,
			Poller: &tele.LongPoller{Timeout: cfg.Bot.Poller},
		}
		log.Info(fmt.Sprintf("Long polling configured with timeout: %s", cfg.Bot.Poller))
	}

	telegramBot, err := tele.NewBot(botSettings)
	if err != nil {
		log.Fatal("Failed to create bot: " + err.Error())
	}
	// Initialize bot services
	services := service.NewServiceManager(*cfg, log, store, telegramBot)
	// Initialize handler
	params := handlers.NewHandlerParams{
		Logger:   log,
		Storage:  store,
		Bot:      telegramBot,
		Cfg:      cfg,
		Services: services,
	}
	handler := handlers.NewHandler(params)

	// Set up routes
	bot.RegisterRoutes(telegramBot, handler, log)
	// Initialize and start expiry worker
	expiryWorker := service.NewExpiryWorker(store, log, telegramBot)
	go expiryWorker.Start()

	log.Info("Bot started successfully! Press Ctrl+C to stop.")

	// Graceful shutdown
	go func() {
		telegramBot.Start()
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Info("Shutting down bot...")

	// Stop expiry worker
	expiryWorker.Stop()

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Stop the bot
	telegramBot.Stop()

	// Wait for context or timeout
	<-ctx.Done()
	log.Info("Bot stopped gracefully")
}
