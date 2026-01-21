package handlers

import (
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/storage"

	telebot "gopkg.in/telebot.v4"
)

// BotHandlers contains all bot command and callback handlers
type Handler struct {
	log     logger.LoggerI
	storage storage.StorageI
	bot     *telebot.Bot
}

// NewHandler creates a new instance of bot handlers
func NewHandler(logger logger.LoggerI, storage storage.StorageI, bot *telebot.Bot) *Handler {
	return &Handler{
		log:     logger,
		storage: storage,
		bot:     bot,
	}
}
