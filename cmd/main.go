package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"telegram-bot-starter/bot"
	"telegram-bot-starter/bot/handlers"
	"telegram-bot-starter/config"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/storage/postgres"

	"github.com/joho/godotenv"
	tele "gopkg.in/telebot.v4"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		panic("Failed to load .env file: " + err.Error())
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

	// Create bot instance
	pref := tele.Settings{
		Token:  cfg.Bot.Token,
		Poller: &tele.LongPoller{Timeout: cfg.Bot.Poller},
	}

	telegramBot, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal("Failed to create bot: " + err.Error())
	}

	// Initialize handler
	handler := handlers.NewHandler(log, store, telegramBot)

	// Set up routes
	bot.SetUpRoutes(telegramBot, handler, log)

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

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Stop the bot
	telegramBot.Stop()

	// Wait for context or timeout
	<-ctx.Done()
	log.Info("Bot stopped gracefully")
}
