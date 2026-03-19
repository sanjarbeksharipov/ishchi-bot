package middleware

import (
	"context"

	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/storage"

	tele "gopkg.in/telebot.v4"
)

// UserSyncMiddleware ensures the user's information (like username, first name, last name)
// is up-to-date on every interaction with the bot.
func UserSyncMiddleware(store storage.StorageI, log logger.LoggerI) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			sender := c.Sender()
			if sender != nil {
				// We don't want to block the ongoing request handling just for a user sync.
				// Run it asynchronously in the background.
				go func() {
					_, err := store.User().GetOrCreateUser(context.Background(), sender.ID, sender.Username, sender.FirstName, sender.LastName)
					if err != nil {
						log.Error("Failed to sync user in middleware", logger.Error(err))
					}
				}()
			}
			return next(c)
		}
	}
}
