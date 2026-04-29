package tasks

import (
	"atlas-buffs/character"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestPoisonTick_SleepTime_RespectsConfiguredInterval(t *testing.T) {
	pt := NewPoisonTick(logrus.New(), 750)
	require.Equal(t, 750*time.Millisecond, pt.SleepTime())
}

func TestPoisonTick_SleepTime_DefaultMillisecondMath(t *testing.T) {
	pt := NewPoisonTick(logrus.New(), 1000)
	require.Equal(t, time.Second, pt.SleepTime())
}

// TestPoisonTick_Run_DoesNotPanicWithNoTenants verifies Run() is safe to invoke
// when there are no registered tenants in Redis. It uses miniredis to stand in
// for the registry's backing store so Run()'s call into
// character.ProcessPoisonTicks reaches a real (but empty) tenant set instead of
// dereferencing a nil registry client.
func TestPoisonTick_Run_DoesNotPanicWithNoTenants(t *testing.T) {
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	character.InitRegistry(client)

	pt := NewPoisonTick(logrus.New(), 1000)
	require.NotPanics(t, func() { pt.Run() })
}
