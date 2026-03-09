package model

import "time"

// MsTime converts a time.Time to a Windows FILETIME-compatible int64 value.
func MsTime(t time.Time) int64 {
	if t.IsZero() {
		return -1
	}
	return t.Unix()*int64(10000000) + int64(116444736000000000)
}
