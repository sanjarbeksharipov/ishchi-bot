package bot

import (
	"fmt"
	"strings"
	"telegram-bot-starter/bot/handlers"
	"telegram-bot-starter/pkg/logger"

	tele "gopkg.in/telebot.v4"
)

func SetUpRoutes(bot *tele.Bot, handler *handlers.Handler, log logger.LoggerI) {
	// Initialize handlers

	// Apply middleware
	// bot.Use(middleware.LoggingMiddleware(log))
	// Register command handlers
	bot.Handle("/start", handler.HandleStart)
	bot.Handle("/help", handler.HandleHelp)
	bot.Handle("/about", handler.HandleAbout)
	bot.Handle("/settings", handler.HandleSettings)

	// Register callback handlers
	bot.Handle(tele.OnCallback, func(c tele.Context) error {
		data := c.Callback().Data
		switch strings.TrimSpace(data) {
		case "help":
			return handler.HandleHelpCallback(c)
		case "about":
			return handler.HandleAboutCallback(c)
		case "settings":
			return handler.HandleSettingsCallback(c)
		case "back":
			return handler.HandleBackCallback(c)
		case "confirm_yes":
			return handler.HandleConfirmYesCallback(c)
		case "confirm_no":
			return handler.HandleConfirmNoCallback(c)
		default:
			fmt.Println(data)
			return c.Respond(&tele.CallbackResponse{Text: "Unknown action"})
		}
	})

	// Register text message handler
	bot.Handle(tele.OnText, handler.HandleText)
}
