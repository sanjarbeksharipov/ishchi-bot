# Telegram Bot Architecture Documentation

## Overview

This project follows a **clean architecture pattern** with clear separation of concerns. The architecture is organized into distinct layers, each with specific responsibilities.

## Architecture Principles

### 1. **Separation of Concerns**
- Each layer has a single, well-defined responsibility
- Business logic is isolated from infrastructure concerns
- Presentation logic is separated from business logic

### 2. **Dependency Rule**
- Dependencies flow inward: Handlers → Services → Storage → Database
- Inner layers never depend on outer layers
- Inner layers define interfaces that outer layers implement

### 3. **Testability**
- Business logic in services can be tested independently
- Storage layer can be mocked for service testing
- Handlers contain minimal logic

## Layer Responsibilities

### **1. Handlers Layer** (`bot/handlers/`)

**Purpose**: Handle user interactions and bot events

**Responsibilities**:
- Receive and parse Telegram updates (messages, callbacks, commands)
- Validate user input format
- Call appropriate service methods
- Format and send responses to users
- Handle Telegram-specific concerns (keyboards, message editing, etc.)

**MUST NOT**:
- ❌ Contain business logic
- ❌ Directly access storage/database
- ❌ Manage transactions
- ❌ Perform complex data transformations

**Example**:
```go
func (h *Handler) HandleBookingConfirm(c tele.Context, jobID int64) error {
    ctx := context.Background()
    userID := c.Sender().ID

    // ✅ Call service for business logic
    booking, err := h.services.Booking().ConfirmBooking(ctx, userID, jobID)
    if err != nil {
        // ✅ Handle error and format user message
        return c.Edit("❌ Xatolik yuz berdi.")
    }

    // ✅ Format and send response
    msg := fmt.Sprintf("✅ Booking confirmed! ID: %d", booking.ID)
    return c.Send(msg)
}
```

### **2. Services Layer** (`service/`)

**Purpose**: Implement business logic and orchestrate operations

**Responsibilities**:
- Implement core business rules and workflows
- Coordinate multiple storage operations
- Manage transactions
- Perform data validation and transformations
- Enforce business constraints
- Handle complex multi-step operations

**MUST NOT**:
- ❌ Know about Telegram-specific details (messages, keyboards, callbacks)
- ❌ Format user-facing messages
- ❌ Directly handle HTTP/Telegram API calls

**Example**:
```go
func (s *bookingService) ConfirmBooking(ctx context.Context, userID, jobID int64) (*models.JobBooking, error) {
    // ✅ Start transaction
    tx, err := s.storage.Transaction().Begin(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer func() {
        if err != nil {
            s.storage.Transaction().Rollback(ctx, tx)
        }
    }()

    // ✅ Business logic: check job availability
    job, err := s.storage.Job().GetByIDForUpdate(ctx, tx, jobID)
    if err != nil {
        return nil, err
    }

    if job.IsFull() {
        return nil, fmt.Errorf("all slots are full")
    }

    // ✅ Business logic: reserve slot atomically
    if err := s.storage.Job().IncrementReservedSlots(ctx, tx, jobID); err != nil {
        return nil, err
    }

    // ✅ Create booking
    booking := &models.JobBooking{
        UserID:    userID,
        JobID:     jobID,
        Status:    models.BookingStatusSlotReserved,
        ExpiresAt: time.Now().Add(3 * time.Minute),
    }

    if err := s.storage.Booking().Create(ctx, tx, booking); err != nil {
        return nil, err
    }

    // ✅ Commit transaction
    if err := s.storage.Transaction().Commit(ctx, tx); err != nil {
        return nil, err
    }

    return booking, nil
}
```

### **3. Storage Layer** (`storage/`)

**Purpose**: Abstract database operations

**Responsibilities**:
- Define storage interfaces
- Implement database queries
- Handle data persistence
- Provide CRUD operations
- Manage database-specific concerns (SQL, connection pools, etc.)

**MUST NOT**:
- ❌ Contain business logic
- ❌ Know about business rules
- ❌ Manage transaction lifecycle (begin/commit) - that's service responsibility

**Example**:
```go
// Interface definition (storage/storage.go)
type BookingRepoI interface {
    Create(ctx context.Context, tx any, booking *models.JobBooking) error
    GetByID(ctx context.Context, id int64) (*models.JobBooking, error)
    GetByIDForUpdate(ctx context.Context, tx any, id int64) (*models.JobBooking, error)
    Update(ctx context.Context, tx any, booking *models.JobBooking) error
}

// Implementation (storage/postgres/booking.go)
func (r *bookingRepo) Create(ctx context.Context, tx any, booking *models.JobBooking) error {
    // ✅ Pure database operation
    query := `INSERT INTO job_bookings (user_id, job_id, status, expires_at) VALUES ($1, $2, $3, $4) RETURNING id`
    return r.db.QueryRow(ctx, query, booking.UserID, booking.JobID, booking.Status, booking.ExpiresAt).Scan(&booking.ID)
}
```

### **4. Models Layer** (`bot/models/`)

**Purpose**: Define data structures and domain entities

**Responsibilities**:
- Define domain models (User, Job, Booking, etc.)
- Define enums and constants
- Include basic domain methods (IsExpired, IsFull, etc.)

**MUST NOT**:
- ❌ Contain business logic
- ❌ Have dependencies on other layers
- ❌ Perform I/O operations

## Service Organization

### Current Services

#### **RegistrationService** (`service/registration.go`)
- Handles user registration flow
- Manages registration drafts
- Validates registration data
- Creates registered users

#### **BookingService** (`service/booking.go`)
- Handles job booking logic
- Manages slot reservations
- Implements idempotency checks
- Handles booking expiration

#### **PaymentService** (`service/payment.go`)
- Handles payment submission
- Approves/rejects payments
- Manages booking status transitions
- Handles user blocking

#### **SenderService** (`service/sender.go`)
- Sends notifications to users
- Broadcasts messages to channels
- Updates channel messages

## Data Flow

### Example: User Books a Job

```
User clicks "Book" button
         ↓
Handler receives callback
         ↓
Handler calls: services.Booking().ConfirmBooking()
         ↓
BookingService:
  1. Begins transaction
  2. Locks job row
  3. Checks availability
  4. Increments reserved slots
  5. Creates booking record
  6. Commits transaction
  7. Returns booking
         ↓
Handler formats success message
         ↓
Handler sends message to user
```

## Transaction Management

### Rules

1. **Services manage transactions**, NOT handlers
2. Use `defer` for rollback safety:
```go
tx, err := s.storage.Transaction().Begin(ctx)
if err != nil {
    return err
}
defer func() {
    if err != nil {
        s.storage.Transaction().Rollback(ctx, tx)
    }
}()
```

3. Always use row-level locking for concurrent operations:
```go
job, err := s.storage.Job().GetByIDForUpdate(ctx, tx, jobID)
```

4. Keep transactions short and focused

## Error Handling

### Service Layer
- Return descriptive errors
- Use `fmt.Errorf` with context
- Don't return user-facing messages

```go
if job.IsFull() {
    return nil, fmt.Errorf("all slots are full")
}
```

### Handler Layer
- Catch service errors
- Map to user-friendly messages
- Handle Telegram-specific errors

```go
booking, err := h.services.Booking().ConfirmBooking(ctx, userID, jobID)
if err != nil {
    if err.Error() == "all slots are full" {
        return c.Edit("❌ Kechirasiz, barcha joylar band bo'lib qoldi!")
    }
    return c.Edit("❌ Xatolik yuz berdi.")
}
```

## Adding New Features

### Step 1: Define Model (if needed)
```go
// bot/models/notification.go
type Notification struct {
    ID        int64
    UserID    int64
    Message   string
    SentAt    time.Time
}
```

### Step 2: Define Storage Interface
```go
// storage/storage.go
type NotificationRepoI interface {
    Create(ctx context.Context, tx any, notif *Notification) error
    GetByUserID(ctx context.Context, userID int64) ([]*Notification, error)
}
```

### Step 3: Implement Storage
```go
// storage/postgres/notification.go
func (r *notificationRepo) Create(ctx context.Context, tx any, notif *Notification) error {
    // Database implementation
}
```

### Step 4: Create Service
```go
// service/notification.go
type NotificationService interface {
    Send(ctx context.Context, userID int64, message string) error
    GetHistory(ctx context.Context, userID int64) ([]*models.Notification, error)
}

func (s *notificationService) Send(ctx context.Context, userID int64, message string) error {
    // Business logic here
}
```

### Step 5: Register Service
```go
// service/service.go
type ServiceManagerI interface {
    // ... other services
    Notification() NotificationService
}

func NewServiceManager(...) *ServiceManager {
    services := &ServiceManager{}
    // ... other services
    services.notificationService = NewNotificationService(cfg, log, storage, services)
    return services
}
```

### Step 6: Use in Handler
```go
// bot/handlers/notification.go
func (h *Handler) HandleNotificationHistory(c tele.Context) error {
    ctx := context.Background()
    
    notifications, err := h.services.Notification().GetHistory(ctx, c.Sender().ID)
    if err != nil {
        return c.Send("❌ Xatolik yuz berdi.")
    }
    
    // Format and send to user
    return c.Send(formatNotifications(notifications))
}
```

## Common Anti-Patterns to Avoid

### ❌ DON'T: Business Logic in Handlers
```go
func (h *Handler) HandleBooking(c tele.Context) error {
    // ❌ BAD: Business logic in handler
    tx, _ := h.storage.Transaction().Begin(ctx)
    job, _ := h.storage.Job().GetByID(ctx, jobID)
    if job.ReservedSlots + job.ConfirmedSlots >= job.RequiredWorkers {
        return c.Send("Full")
    }
    // ... more business logic
}
```

### ✅ DO: Business Logic in Services
```go
func (h *Handler) HandleBooking(c tele.Context) error {
    // ✅ GOOD: Delegate to service
    booking, err := h.services.Booking().ConfirmBooking(ctx, userID, jobID)
    if err != nil {
        return h.handleBookingError(c, err)
    }
    return c.Send(formatBookingSuccess(booking))
}
```

### ❌ DON'T: Telegram Logic in Services
```go
func (s *bookingService) ConfirmBooking(...) error {
    // ❌ BAD: Telegram-specific code in service
    _, err := s.bot.Send(user, "Booking confirmed!")
    return err
}
```

### ✅ DO: Return Data, Let Handler Format
```go
func (s *bookingService) ConfirmBooking(...) (*models.JobBooking, error) {
    // ✅ GOOD: Return domain object
    return booking, nil
}

func (h *Handler) HandleBooking(c tele.Context) error {
    booking, _ := h.services.Booking().ConfirmBooking(...)
    // ✅ Handler formats message
    return c.Send(fmt.Sprintf("✅ Booking %d confirmed!", booking.ID))
}
```

## Testing Strategy

### Service Layer Testing
- Mock storage interface
- Test business logic independently
- Test transaction handling
- Test error cases

### Handler Layer Testing
- Mock services
- Test message formatting
- Test error handling
- Test user flow

## Code Review Checklist

- [ ] Is business logic in services, not handlers?
- [ ] Are transactions managed by services?
- [ ] Do handlers only handle Telegram concerns?
- [ ] Are storage methods pure database operations?
- [ ] Are errors properly handled at each layer?
- [ ] Is the code testable?
- [ ] Are interfaces used for dependencies?

## Summary

**Remember**:
- Handlers = Telegram interface (receive, format, send)
- Services = Business logic (validate, coordinate, transform)
- Storage = Data persistence (query, save, update)

**The Golden Rule**: 
> If you're writing business logic, it belongs in a service.
> If you're formatting a message, it belongs in a handler.
> If you're writing SQL, it belongs in storage.

## Migration Guide

When you find code that violates these principles:

1. **Identify the violation**: Is it business logic in handlers? Telegram logic in services?
2. **Create the service** if it doesn't exist
3. **Move the logic** to the appropriate service method
4. **Update the handler** to call the service
5. **Test** the changes

## Additional Resources

- See `service/booking.go` for service implementation example
- See `bot/handlers/booking.go` for handler implementation example
- See `storage/postgres/booking.go` for storage implementation example
