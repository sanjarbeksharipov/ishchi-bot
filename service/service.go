package service

import (
	"telegram-bot-starter/config"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/storage"

	tele "gopkg.in/telebot.v4"
)

// ServiceManagerI defines the main service interface
type ServiceManagerI interface {
	Registration() RegistrationService
	Sender() *SenderService
	Booking() BookingService
	Payment() PaymentService
}

// ServiceManager holds all service instances
type ServiceManager struct {
	registrationService RegistrationService
	senderService       *SenderService
	bookingService      BookingService
	paymentService      PaymentService
}

// NewServiceManager initializes and returns a new ServiceManager
func NewServiceManager(cfg config.Config, log logger.LoggerI, storage storage.StorageI, bot *tele.Bot) *ServiceManager {
	services := &ServiceManager{}

	services.registrationService = NewRegistrationService(cfg, log, storage, services)
	services.senderService = NewSenderService(cfg, log, bot, storage, services)
	services.bookingService = NewBookingService(cfg, log, storage, services)
	services.paymentService = NewPaymentService(cfg, log, storage, services)

	return services
}

// Registration returns the registration service
func (s *ServiceManager) Registration() RegistrationService {
	return s.registrationService
}

// Sender returns the sender service
func (s *ServiceManager) Sender() *SenderService {
	return s.senderService
}

// Booking returns the booking service
func (s *ServiceManager) Booking() BookingService {
	return s.bookingService
}

// Payment returns the payment service
func (s *ServiceManager) Payment() PaymentService {
	return s.paymentService
}
