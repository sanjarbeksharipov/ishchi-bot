package handlers

import (
	"context"
	"strings"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/pkg/keyboards"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/pkg/messages"

	tele "gopkg.in/telebot.v4"
)

// HandleStart handles the /start command
func (h *Handler) HandleStart(c tele.Context) error {
	ctx := context.Background()
	user := c.Sender()

	// Get or create user in storage
	_, err := h.storage.User().GetOrCreateUser(ctx, user.ID, user.Username, user.FirstName, user.LastName)
	if err != nil {
		h.log.Error("Failed to get/create user", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Check if this is an admin
	if h.IsAdmin(user.ID) {
		return c.Send(messages.MsgWelcome, keyboards.MainMenuKeyboard())
	}

	// For regular users, start/continue registration flow
	return h.HandleRegistrationStart(c)
}

// HandleHelp handles the /help command
func (h *Handler) HandleHelp(c tele.Context) error {
	return c.Send(messages.MsgHelp, keyboards.BackKeyboard())
}

// HandleAbout handles the /about command
func (h *Handler) HandleAbout(c tele.Context) error {
	return c.Send(messages.MsgAbout, keyboards.BackKeyboard())
}

// HandleSettings handles the /settings command
func (h *Handler) HandleSettings(c tele.Context) error {
	return c.Send(messages.MsgSettings, keyboards.BackKeyboard())
}

// HandleText handles regular text messages
func (h *Handler) HandleText(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	text := strings.TrimSpace(c.Text())

	// Get or create user
	user, err := h.storage.User().GetOrCreateUser(ctx, sender.ID, sender.Username, sender.FirstName, sender.LastName)
	if err != nil {
		h.log.Error("Failed to get/create user", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Handle cancel button from reply keyboard
	if text == "❌ Bekor qilish" {
		return h.HandleCancelRegistration(c)
	}

	// Check if user is in registration flow
	if h.IsInRegistrationFlow(user.State) {
		regState := h.GetRegistrationState(user.State)
		return h.HandleRegistrationTextInput(c, regState)
	}

	// Check if user is in job creation/editing flow (admin only)
	if h.IsAdmin(sender.ID) && (strings.HasPrefix(string(user.State), "creating_job_") || strings.HasPrefix(string(user.State), "editing_job_")) {
		return h.HandleAdminTextInput(c, user)
	}

	// Default: check user state
	switch user.State {
	case models.StateIdle:
		// Echo the message back with a prefix
		response := "You said: " + c.Text()
		return c.Send(response)
	default:
		response := "You said: " + c.Text()
		return c.Send(response)
	}
}

// HandleContact handles contact sharing messages
func (h *Handler) HandleContact(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()

	// Get user
	user, err := h.storage.User().GetByID(ctx, sender.ID)
	if err != nil {
		h.log.Error("Failed to get user", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Check if user is in registration phone state
	if user.State == models.UserState(models.RegStatePhone) {
		return h.HandleRegistrationContact(c)
	}

	return nil
}

// HandlePhoto handles photo messages
func (h *Handler) HandlePhoto(c tele.Context) error {
	// Passport photo step has been removed from registration
	// No special handling needed for photos

	return nil
}
