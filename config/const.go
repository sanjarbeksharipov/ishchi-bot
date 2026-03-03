package config

import "time"

// Timezone is the application timezone (Uzbekistan, UTC+5)
var Timezone = time.FixedZone("Asia/Tashkent", 5*60*60)

// NowLocal returns current time in the configured timezone
func NowLocal() time.Time {
	return time.Now().In(Timezone)
}
