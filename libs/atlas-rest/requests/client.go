package requests

import (
	"net/http"
	"os"
	"time"
)

var DefaultTimeout = 10 * time.Second

var client = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
}

func init() {
	if v := os.Getenv("HTTP_CLIENT_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			DefaultTimeout = d
		}
	}
}
