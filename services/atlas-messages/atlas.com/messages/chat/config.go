package chat

import (
	"os"
	"strconv"
	"sync"
)

const (
	envRetentionSeconds = "CHAT_CAPTURE_RETENTION_SECONDS"
	envMaxLines         = "CHAT_CAPTURE_MAX_LINES"

	defaultRetentionSeconds = 900
	defaultMaxLines         = 200
)

var (
	configOnce       sync.Once
	retentionSeconds int
	maxLines         int
)

func envInt(name string, def int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return def
	}
	return v
}

func loadConfig() {
	configOnce.Do(func() {
		retentionSeconds = envInt(envRetentionSeconds, defaultRetentionSeconds)
		maxLines = envInt(envMaxLines, defaultMaxLines)
	})
}
