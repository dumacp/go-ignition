package utils

import "time"

// Get uptime duration
func Uptime() (time.Duration, error) {
	return uptime()
}
