package main

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestLeaderEnabled_DefaultsTrue(t *testing.T) {
	t.Setenv("MONSTER_LEADER_ELECTION_ENABLED", "")
	require.True(t, leaderEnabled(logrus.New()))
}

func TestLeaderEnabled_ParsesFalse(t *testing.T) {
	t.Setenv("MONSTER_LEADER_ELECTION_ENABLED", "false")
	require.False(t, leaderEnabled(logrus.New()))
}

func TestLeaderEnabled_BadValueWarnsAndReturnsTrue(t *testing.T) {
	t.Setenv("MONSTER_LEADER_ELECTION_ENABLED", "maybe")
	require.True(t, leaderEnabled(logrus.New()))
}

func TestLeaderTTL_DefaultsTo30s(t *testing.T) {
	t.Setenv("MONSTER_LEADER_TTL", "")
	require.Equal(t, 30*time.Second, leaderTTL(logrus.New()))
}

func TestLeaderTTL_OutOfRangeFallsBack(t *testing.T) {
	t.Setenv("MONSTER_LEADER_TTL", "1s")
	require.Equal(t, 30*time.Second, leaderTTL(logrus.New()))
	t.Setenv("MONSTER_LEADER_TTL", "10m")
	require.Equal(t, 30*time.Second, leaderTTL(logrus.New()))
}

func TestLeaderTTL_ValidValueAccepted(t *testing.T) {
	t.Setenv("MONSTER_LEADER_TTL", "60s")
	require.Equal(t, 60*time.Second, leaderTTL(logrus.New()))
}

func TestLeaderRefresh_DefaultsToTTLOver3(t *testing.T) {
	t.Setenv("MONSTER_LEADER_TTL", "30s")
	t.Setenv("MONSTER_LEADER_REFRESH", "")
	require.Equal(t, 10*time.Second, leaderRefresh(logrus.New(), 30*time.Second))
}

func TestLeaderRefresh_OutOfRangeFallsBack(t *testing.T) {
	t.Setenv("MONSTER_LEADER_REFRESH", "0s")
	require.Equal(t, 10*time.Second, leaderRefresh(logrus.New(), 30*time.Second))
	t.Setenv("MONSTER_LEADER_REFRESH", "20s") // > TTL/2 = 15s
	require.Equal(t, 10*time.Second, leaderRefresh(logrus.New(), 30*time.Second))
}

func TestLeaderBackoff_DefaultsTo5s(t *testing.T) {
	t.Setenv("MONSTER_LEADER_BACKOFF", "")
	require.Equal(t, 5*time.Second, leaderBackoff(logrus.New()))
}

func TestLeaderBackoff_OutOfRangeFallsBack(t *testing.T) {
	t.Setenv("MONSTER_LEADER_BACKOFF", "0s")
	require.Equal(t, 5*time.Second, leaderBackoff(logrus.New()))
	t.Setenv("MONSTER_LEADER_BACKOFF", "5m")
	require.Equal(t, 5*time.Second, leaderBackoff(logrus.New()))
}
