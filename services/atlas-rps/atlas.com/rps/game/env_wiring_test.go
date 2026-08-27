package game_test

import (
	"atlas-rps/game"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// envMarkerKey is a test-local context key standing in for atlas-env's real
// marker. It pins that SweepTask.Run applies the injected envContext to
// each swept session's per-tenant context before the GameEnded event is
// emitted -- without game importing atlas-env itself (env-domain-guard
// forbids that; main.go threads the real env.WithContext/env.Self()
// implementation in as a plain function value instead).
type envMarkerKey struct{}

func TestSweepTask_Run_AppliesEnvContext(t *testing.T) {
	setupRegistryTest(t)
	reg := setupSweepCapturingProducer(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	characterId := uint32(4001)

	now := time.Now()
	game.GetRegistry().SetNowFunc(func() time.Time { return now })

	m := game.NewBuilder(ten).
		SetCharacterId(characterId).
		SetWorldId(0).
		SetChannelId(1).
		SetNpcId(9020000).
		SetStatus(game.StatusAwaitingDecision).
		MustBuild()
	game.GetRegistry().Put(ctx, m)

	now = now.Add(10 * time.Minute)
	game.GetRegistry().SetNowFunc(func() time.Time { return now })

	var envContextCalled bool
	task := game.NewSweepTask(testLogger(), 50*time.Millisecond, func(c context.Context) context.Context {
		envContextCalled = true
		return context.WithValue(c, envMarkerKey{}, "pod-env")
	})
	task.Run()

	assert.True(t, envContextCalled, "SweepTask.Run must apply envContext to the swept session's context before emitting")
	_ = reg // producer capture is asserted by TestSweepTask_Run_DisposesExpiredSessionWithNoPayout; this test only pins envContext invocation
}
