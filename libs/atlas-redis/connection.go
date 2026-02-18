package redis

import (
	"os"

	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

const defaultRedisURL = "localhost:6379"

func Connect(l logrus.FieldLogger) *goredis.Client {
	addr := os.Getenv("REDIS_URL")
	if addr == "" {
		addr = defaultRedisURL
	}

	password := os.Getenv("REDIS_PASSWORD")

	opts := &goredis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	}

	client := goredis.NewClient(opts)
	l.Infof("Connecting to Redis at [%s].", addr)
	return client
}
