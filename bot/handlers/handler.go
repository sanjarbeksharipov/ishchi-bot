package handlers

import (
	"telegram-bot-starter/config"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/service"
	"telegram-bot-starter/storage"

	telebot "gopkg.in/telebot.v4"
)

// Handler contains all bot command and callback handlers
type Handler struct {
	log      logger.LoggerI
	storage  storage.StorageI
	bot      *telebot.Bot
	cfg      *config.Config
	services service.ServiceManagerI
}
type NewHandlerParams struct {
	Logger   logger.LoggerI
	Storage  storage.StorageI
	Bot      *telebot.Bot
	Cfg      *config.Config
	Services service.ServiceManagerI
}

// NewHandler creates a new instance of bot handlers
func NewHandler(params NewHandlerParams) *Handler {

	h := &Handler{
		log:      params.Logger,
		storage:  params.Storage,
		bot:      params.Bot,
		cfg:      params.Cfg,
		services: params.Services,
	}
	return h
}
