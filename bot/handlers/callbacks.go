package handlers

import (
	"telegram-bot-starter/pkg/keyboards"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/pkg/messages"

	tele "gopkg.in/telebot.v4"
)

// HandleHelpCallback handles the help button callback
func (h *Handler) HandleHelpCallback(c tele.Context) error {
	if err := c.Respond(); err != nil {
		h.log.Error("Failed to respond to callback", logger.Error(err))
	}
	return c.Edit(messages.MsgHelp, keyboards.BackKeyboard())
}

// HandleAboutCallback handles the about button callback
func (h *Handler) HandleAboutCallback(c tele.Context) error {
	if err := c.Respond(); err != nil {
		h.log.Error("Failed to respond to callback", logger.Error(err))
	}
	return c.Edit(messages.MsgAbout, keyboards.BackKeyboard())
}

// HandleSettingsCallback handles the settings button callback
func (h *Handler) HandleSettingsCallback(c tele.Context) error {
	if err := c.Respond(); err != nil {
		h.log.Error("Failed to respond to callback", logger.Error(err))
	}
	return c.Edit(messages.MsgSettings, keyboards.BackKeyboard())
}

// HandleBackCallback handles the back button callback
func (h *Handler) HandleBackCallback(c tele.Context) error {
	if err := c.Respond(); err != nil {
		h.log.Error("Failed to respond to callback", logger.Error(err))
	}
	return c.Edit(messages.MsgWelcome, keyboards.MainMenuKeyboard())
}

// HandleConfirmYesCallback handles confirmation yes callback
func (h *Handler) HandleConfirmYesCallback(c tele.Context) error {
	if err := c.Respond(&tele.CallbackResponse{Text: "✅ Confirmed!"}); err != nil {
		h.log.Error("Failed to respond to callback", logger.Error(err))
	}
	return c.Edit("Action confirmed!", keyboards.BackKeyboard())
}

// HandleConfirmNoCallback handles confirmation no callback
func (h *Handler) HandleConfirmNoCallback(c tele.Context) error {
	if err := c.Respond(&tele.CallbackResponse{Text: "❌ Cancelled"}); err != nil {
		h.log.Error("Failed to respond to callback", logger.Error(err))
	}
	return c.Edit("Action cancelled.", keyboards.BackKeyboard())
}
