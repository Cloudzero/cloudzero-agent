package utils

import "time"

type TimeProvider interface {
	GetCurrentTime() time.Time
}

type Clock struct{}

func (c *Clock) GetCurrentTime() time.Time {
	return time.Now().UTC()
}

// formats a time.Time value to the ISO 8601 format
func FormatForStorage(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}
