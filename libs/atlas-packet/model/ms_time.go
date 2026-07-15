package model

import (
	"encoding/binary"
	"time"
)

// MsTime converts a time.Time to a Windows FILETIME-compatible int64 value.
func MsTime(t time.Time) int64 {
	if t.IsZero() {
		return -1
	}
	return t.Unix()*int64(10000000) + int64(116444736000000000)
}

// MsTimeBytes converts a time.Time to its little-endian 8-byte FILETIME wire
// form (the shape DecodeBuffer(8) FILETIME fields expect, e.g. the MTS ITCITEM
// ftITCDateExpired). A zero time yields the MsTime(-1) sentinel bytes.
func MsTimeBytes(t time.Time) [8]byte {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], uint64(MsTime(t)))
	return b
}

// FromMsTime converts a Windows FILETIME-compatible int64 value back to time.Time.
func FromMsTime(v int64) time.Time {
	if v == -1 {
		return time.Time{}
	}
	return time.Unix((v-116444736000000000)/10000000, 0)
}
