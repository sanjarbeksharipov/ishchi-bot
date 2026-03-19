package bot

import (
	"telegram-bot-starter/bot/handlers"
	"telegram-bot-starter/bot/middleware"
	"telegram-bot-starter/config"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/storage"

	tele "gopkg.in/telebot.v4"
)

func RegisterRoutes(bot *tele.Bot, handler *handlers.Handler, log logger.LoggerI, cfg *config.Config, store storage.StorageI) *middleware.RateLimiter {
	// Apply middleware
	// Recovery middleware MUST be first — it catches panics from all subsequent handlers/middleware.
	// Without it, a panic kills the polling goroutine silently (container stays up, bot stops responding).
	bot.Use(middleware.RecoveryMiddleware(log))

	// Ensure user details are synchronized on every interaction
	bot.Use(middleware.UserSyncMiddleware(store, log))

	// Apply rate limiter middleware
	rateLimiter := middleware.NewRateLimiter(cfg, log)
	bot.Use(rateLimiter.Middleware())

	// Register command handlers
	bot.Handle("/start", handler.HandleStart)
	bot.Handle("/help", handler.HandleHelp)
	bot.Handle("/about", handler.HandleAbout)
	bot.Handle("/settings", handler.HandleSettings)
	bot.Handle("/admin", handler.HandleAdminPanel)

	// Register callback handler (routing lives in handlers/callback_router.go)
	bot.Handle(tele.OnCallback, handler.HandleCallback)

	// Register text message handler
	bot.Handle(tele.OnText, handler.HandleText)

	// Register contact handler (for phone sharing)
	bot.Handle(tele.OnContact, handler.HandleContact)

	// Register photo handler (for payment proofs)
	bot.Handle(tele.OnPhoto, handler.HandlePhoto)

	// Register location handler (for job locations)
	bot.Handle(tele.OnLocation, handler.HandleLocation)

	return rateLimiter
}
