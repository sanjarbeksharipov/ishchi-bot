package middleware

import (
	"fmt"

	"telegram-bot-starter/pkg/logger"

	tele "gopkg.in/telebot.v4"
)

// LoggingMiddleware logs all incoming updates
func LoggingMiddleware(log logger.LoggerI) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			user := c.Sender()
			var username string
			if user != nil {
				if user.Username != "" {
					username = "@" + user.Username
				} else {
					username = user.FirstName
				}
			}

			if c.Message() != nil {
				log.Info(fmt.Sprintf("Message from %s (ID: %d): %s", username, user.ID, c.Text()))
			} else if c.Callback() != nil {
				log.Info(fmt.Sprintf("Callback from %s (ID: %d): %s", username, user.ID, c.Callback().Data))
			}

			return next(c)
		}
	}
}
