package model

import "time"

// MsTime converts a time.Time to a Windows FILETIME-compatible int64 value.
func MsTime(t time.Time) int64 {
	if t.IsZero() {
		return -1
	}
	return t.Unix()*int64(10000000) + int64(116444736000000000)
}

// FromMsTime converts a Windows FILETIME-compatible int64 value back to time.Time.
func FromMsTime(v int64) time.Time {
	if v == -1 {
		return time.Time{}
	}
	return time.Unix((v-116444736000000000)/10000000, 0)
}
