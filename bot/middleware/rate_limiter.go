package middleware

import (
	"fmt"
	"sync"
	"time"

	"telegram-bot-starter/config"
	"telegram-bot-starter/pkg/logger"

	tele "gopkg.in/telebot.v4"
)

// userBucket tracks request timestamps for a single user using a sliding window.
type userBucket struct {
	mu         sync.Mutex
	timestamps []time.Time
}

// RateLimiter provides per-user rate limiting for Telegram bot updates.
type RateLimiter struct {
	maxRequests int           // max requests allowed within the window
	window      time.Duration // sliding window duration
	adminIDs    []int64       // admin user IDs exempt from limiting
	log         logger.LoggerI

	mu      sync.RWMutex
	buckets map[int64]*userBucket

	stopCleanup chan struct{}
}

// NewRateLimiter creates a rate limiter from config.
func NewRateLimiter(cfg *config.Config, log logger.LoggerI) *RateLimiter {
	maxReq := cfg.Bot.RateLimitMaxRequests
	if maxReq <= 0 {
		maxReq = 30 // default: 30 requests
	}
	window := cfg.Bot.RateLimitWindow
	if window <= 0 {
		window = 60 * time.Second // default: per 60 seconds
	}

	rl := &RateLimiter{
		maxRequests: maxReq,
		window:      window,
		adminIDs:    cfg.Bot.AdminIDs,
		log:         log,
		buckets:     make(map[int64]*userBucket),
		stopCleanup: make(chan struct{}),
	}

	// Background goroutine to evict stale entries and prevent memory leaks
	go rl.cleanupLoop()

	return rl
}

// Stop terminates the background cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stopCleanup)
}

// Middleware returns a telebot middleware that enforces the rate limit.
func (rl *RateLimiter) Middleware() tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			sender := c.Sender()
			if sender == nil {
				return next(c)
			}

			userID := sender.ID

			// Admins are exempt from rate limiting
			if rl.isAdmin(userID) {
				return next(c)
			}

			if !rl.allow(userID) {
				rl.log.Warn(fmt.Sprintf("Rate limit exceeded for user %d", userID))

				return nil
			}

			return next(c)
		}
	}
}

// allow checks if the user is within their rate limit and records the request.
func (rl *RateLimiter) allow(userID int64) bool {
	bucket := rl.getBucket(userID)

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Remove expired timestamps (sliding window)
	valid := 0
	for _, t := range bucket.timestamps {
		if t.After(cutoff) {
			bucket.timestamps[valid] = t
			valid++
		}
	}
	bucket.timestamps = bucket.timestamps[:valid]

	if len(bucket.timestamps) >= rl.maxRequests {
		return false
	}

	bucket.timestamps = append(bucket.timestamps, now)
	return true
}

// getBucket returns or creates the bucket for a user.
func (rl *RateLimiter) getBucket(userID int64) *userBucket {
	rl.mu.RLock()
	b, ok := rl.buckets[userID]
	rl.mu.RUnlock()
	if ok {
		return b
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()
	// Double-check after acquiring write lock
	if b, ok = rl.buckets[userID]; ok {
		return b
	}
	b = &userBucket{}
	rl.buckets[userID] = b
	return b
}

// isAdmin checks whether the given user ID is in the admin list.
func (rl *RateLimiter) isAdmin(userID int64) bool {
	for _, id := range rl.adminIDs {
		if id == userID {
			return true
		}
	}
	return false
}

// cleanupLoop periodically removes buckets for users with no recent activity.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.evictStale()
		case <-rl.stopCleanup:
			return
		}
	}
}

// evictStale removes user buckets whose timestamps are all outside the window.
func (rl *RateLimiter) evictStale() {
	cutoff := time.Now().Add(-rl.window)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	for userID, bucket := range rl.buckets {
		bucket.mu.Lock()
		allExpired := true
		for _, t := range bucket.timestamps {
			if t.After(cutoff) {
				allExpired = false
				break
			}
		}
		bucket.mu.Unlock()

		if allExpired {
			delete(rl.buckets, userID)
		}
	}
}
