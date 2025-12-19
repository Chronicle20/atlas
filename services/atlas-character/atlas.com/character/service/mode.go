package service

import "os"

const (
	ReadOnly = "READ_ONLY"
	Mixed    = "MIXED"
)

func GetMode() string {
	val, ok := os.LookupEnv("SERVICE_MODE")
	if ok {
		return val
	}
	return Mixed
}
