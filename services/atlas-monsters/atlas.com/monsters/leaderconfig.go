package main

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	envLeaderEnabled = "MONSTER_LEADER_ELECTION_ENABLED"
	envLeaderTTL     = "MONSTER_LEADER_TTL"
	envLeaderRefresh = "MONSTER_LEADER_REFRESH"
	envLeaderBackoff = "MONSTER_LEADER_BACKOFF"

	defaultLeaderTTL     = 30 * time.Second
	defaultLeaderRefresh = 10 * time.Second
	defaultLeaderBackoff = 5 * time.Second
)

func leaderEnabled(l logrus.FieldLogger) bool {
	v := strings.TrimSpace(os.Getenv(envLeaderEnabled))
	if v == "" {
		return true
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		l.Warnf("[%s] value %q not a boolean; defaulting to true.", envLeaderEnabled, v)
		return true
	}
	return b
}

func leaderTTL(l logrus.FieldLogger) time.Duration {
	return parseDurationInRange(l, envLeaderTTL, defaultLeaderTTL, 5*time.Second, 5*time.Minute)
}

func leaderRefresh(l logrus.FieldLogger, ttl time.Duration) time.Duration {
	def := ttl / 3
	if def < time.Second {
		def = time.Second
	}
	return parseDurationInRange(l, envLeaderRefresh, def, time.Second, ttl/2)
}

func leaderBackoff(l logrus.FieldLogger) time.Duration {
	return parseDurationInRange(l, envLeaderBackoff, defaultLeaderBackoff, time.Second, time.Minute)
}

func parseDurationInRange(l logrus.FieldLogger, env string, def, lo, hi time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(env))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		l.Warnf("[%s] value %q not a duration; using default %s.", env, v, def)
		return def
	}
	if d < lo || d > hi {
		l.Warnf("[%s] value %s out of range [%s, %s]; using default %s.", env, d, lo, hi, def)
		return def
	}
	return d
}
