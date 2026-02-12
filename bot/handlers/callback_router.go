package handlers

import (
	"fmt"
	"strings"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/pkg/keyboards"

	tele "gopkg.in/telebot.v4"
)

type callbackFunc func(c tele.Context, params string) error

type callbackRoute struct {
	prefix  string
	handler callbackFunc
}

// It first checks the static (exact-match) table, then falls through to the dynamic (prefix-based) routes.
func (h *Handler) HandleCallback(c tele.Context) error {
	data := strings.TrimSpace(c.Callback().Data)

	// 1. Static callbacks — exact match
	if handler, ok := h.staticCallbacks()[data]; ok {
		return handler(c)
	}

	// 2. Dynamic callbacks — ordered prefix match
	// NOTE: Uses a slice (not map) to guarantee deterministic matching order.
	// If two prefixes overlap (e.g. "job_" and "job_detail_"), the longer/more
	// specific one must come first.
	for _, route := range h.dynamicCallbacks() {
		if params, ok := strings.CutPrefix(data, route.prefix); ok {
			return route.handler(c, params)
		}
	}

	// 3. Unknown callback
	h.log.Warn(fmt.Sprintf("Unknown callback data: %s", data))
	return c.Respond(&tele.CallbackResponse{Text: "Unknown action"})
}

// Static callbacks (exact string match)
func (h *Handler) staticCallbacks() map[string]tele.HandlerFunc {
	return map[string]tele.HandlerFunc{
		// Navigation
		"help":     h.HandleHelpCallback,
		"about":    h.HandleAboutCallback,
		"settings": h.HandleSettingsCallback,
		"back":     h.HandleBackCallback,

		// Confirmation
		"confirm_yes": h.HandleConfirmYesCallback,
		"confirm_no":  h.HandleConfirmNoCallback,

		// Admin
		"admin_menu":          h.HandleAdminPanel,
		"admin_create_job":    h.HandleCreateJob,
		"admin_job_list":      h.HandleJobList,
		"cancel_job_creation": h.HandleCancelJobCreation,
		"skip_field":          h.HandleSkipField,

		// Registration
		"reg_accept_offer":     h.HandleAcceptOffer,
		"reg_decline_offer":    h.HandleDeclineOffer,
		"reg_continue":         h.HandleContinueRegistration,
		"reg_restart":          h.HandleRestartRegistration,
		"reg_confirm":          h.HandleConfirmRegistration,
		"reg_edit":             h.HandleEditRegistration,
		"reg_cancel":           h.HandleCancelRegistration,
		"reg_back_to_confirm":  h.HandleBackToConfirm,
		"reg_edit_full_name":   func(c tele.Context) error { return h.HandleEditField(c, models.EditFieldFullName) },
		"reg_edit_phone":       func(c tele.Context) error { return h.HandleEditField(c, models.EditFieldPhone) },
		"reg_edit_age":         func(c tele.Context) error { return h.HandleEditField(c, models.EditFieldAge) },
		"reg_edit_body_params": func(c tele.Context) error { return h.HandleEditField(c, models.EditFieldBodyParams) },

		// Booking
		"book_cancel": func(c tele.Context) error { return c.Edit("❌ Bekor qilindi.", keyboards.BackKeyboard()) },

		// User
		"user_my_jobs": h.HandleUserMyJobs,
		"user_profile": h.HandleUserProfile,

		// Profile editing
		"edit_profile_full_name":   func(c tele.Context) error { return h.HandleEditProfileField(c, "full_name") },
		"edit_profile_phone":       func(c tele.Context) error { return h.HandleEditProfileField(c, "phone") },
		"edit_profile_age":         func(c tele.Context) error { return h.HandleEditProfileField(c, "age") },
		"edit_profile_body_params": func(c tele.Context) error { return h.HandleEditProfileField(c, "body_params") },
	}
}

// Dynamic callbacks (prefix match, raw params passed to handler)
// Order matters: more specific prefixes must come before shorter overlapping ones.
func (h *Handler) dynamicCallbacks() []callbackRoute {
	return []callbackRoute{
		// Admin — job management
		{"job_detail_", h.HandleJobDetail},
		{"edit_job_", h.HandleEditJobField},
		{"job_status_", h.HandleChangeJobStatus},
		{"publish_job_", h.HandlePublishJob},
		{"delete_channel_msg_", h.HandleDeleteChannelMessage},
		{"delete_job_", h.HandleDeleteJob},
		{"view_job_bookings_", h.HandleViewJobBookings},

		// User — booking
		{"book_confirm_", h.HandleBookingConfirm},
		{"start_reg_job_", h.HandleStartRegistrationForJob},

		// Admin — payment approval
		{"approve_payment_", h.HandleApprovePayment},
		{"reject_payment_", h.HandleRejectPayment},
		{"block_user_", h.HandleBlockUser},

		// Pagination
		{"users_page_", h.HandleUsersListPage},
	}
}
