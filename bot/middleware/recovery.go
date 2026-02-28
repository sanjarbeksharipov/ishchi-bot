package middleware

import (
	"fmt"
	"runtime/debug"

	"telegram-bot-starter/pkg/logger"

	tele "gopkg.in/telebot.v4"
)

// RecoveryMiddleware catches panics in handlers, logs the stack trace,
// and returns an error instead of crashing the bot's polling goroutine.
//
// Without this, a panic in any handler kills the polling loop silently —
// the container stays alive but the bot stops responding to messages.
func RecoveryMiddleware(log logger.LoggerI) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					stack := string(debug.Stack())

					// Gather context for the log entry
					var userID int64
					var username string
					var callbackData string
					var messageText string

					if sender := c.Sender(); sender != nil {
						userID = sender.ID
						username = sender.Username
					}
					if cb := c.Callback(); cb != nil {
						callbackData = cb.Data
					}
					if msg := c.Message(); msg != nil {
						messageText = msg.Text
						if len(messageText) > 100 {
							messageText = messageText[:100] + "..."
						}
					}

					log.Error(fmt.Sprintf("PANIC RECOVERED: %v", r),
						logger.Any("user_id", userID),
						logger.Any("username", username),
						logger.Any("callback_data", callbackData),
						logger.Any("message_text", messageText),
						logger.Any("stack_trace", stack),
					)

					// Return error so telebot's OnError handler can also process it
					err = fmt.Errorf("panic recovered: %v", r)
				}
			}()

			return next(c)
		}
	}
}
