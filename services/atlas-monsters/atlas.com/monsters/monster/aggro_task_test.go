package monster

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type recordedEmit struct {
	Topic                 string
	Type                  string
	ControllerCharacterId uint32
	ControllerHasAggro    bool
}

func newAggroTaskWithRecorder(t *testing.T, bossIds map[uint32]bool) (*MonsterAggroDecayTask, *[]recordedEmit, *int) {
	t.Helper()
	var events []recordedEmit
	bossCalls := 0
	tk := &MonsterAggroDecayTask{
		l:        logrus.New(),
		ctx:      context.Background(),
		interval: AggroSweepInterval,
		bossLookupFn: func(monsterId uint32) bool {
			bossCalls++
			return bossIds[monsterId]
		},
		emit: func(_ tenant.Model, topic string, provider model.Provider[[]kafka.Message]) error {
			msgs, err := provider()
			if err != nil {
				t.Fatalf("provider err: %v", err)
			}
			for _, m := range msgs {
				var env struct {
					Type string `json:"type"`
					Body struct {
						ControllerCharacterId uint32 `json:"controllerCharacterId"`
						ControllerHasAggro    bool   `json:"controllerHasAggro"`
					} `json:"body"`
				}
				if err := json.Unmarshal(m.Value, &env); err != nil {
					t.Fatalf("decode: %v", err)
				}
				events = append(events, recordedEmit{
					Topic:                 topic,
					Type:                  env.Type,
					ControllerCharacterId: env.Body.ControllerCharacterId,
					ControllerHasAggro:    env.Body.ControllerHasAggro,
				})
			}
			return nil
		},
	}
	return tk, &events, &bossCalls
}

// TestAggroDecayTaskFullClearEmitsAggroChanged seeds a non-boss monster with a
// controller and active aggro, fast-forwards the wall-clock past the idle
// threshold, runs Run(), and asserts AGGRO_CHANGED is emitted with the
// existing controller id and controllerHasAggro=false. The controller itself
// is NOT cleared — losing aggro is not the same as losing control.
func TestAggroDecayTaskFullClearEmitsAggroChanged(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
	if _, err := r.ControlMonster(ten, m.UniqueId(), 42); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}
	// First-hit on a controlled monster flips controllerHasAggro true.
	if _, err := r.ApplyDamage(ten, 1, 1, m.UniqueId(), 0); err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}

	tk, events, _ := newAggroTaskWithRecorder(t, nil /* no bosses */)
	// Override now to be far in the future, satisfying the idle threshold.
	tk.nowFn = func() int64 { return AggroIdleThresholdMs + 1_000 }
	tk.Run()

	if len(*events) != 1 {
		t.Fatalf("expected 1 event, got %d (%+v)", len(*events), *events)
	}
	if (*events)[0].Type != EventMonsterStatusAggroChanged {
		t.Errorf("type=%s, want AGGRO_CHANGED", (*events)[0].Type)
	}
	if (*events)[0].ControllerCharacterId != 42 {
		t.Errorf("ControllerCharacterId=%d, want 42", (*events)[0].ControllerCharacterId)
	}
	if (*events)[0].ControllerHasAggro {
		t.Errorf("ControllerHasAggro=%v, want false", (*events)[0].ControllerHasAggro)
	}

	// And confirm Redis state still has the controller intact.
	got, _ := r.GetMonster(ten, m.UniqueId())
	if got.ControlCharacterId() != 42 {
		t.Errorf("controller cleared by decay (want 42, got %d)", got.ControlCharacterId())
	}
	if got.ControllerHasAggro() {
		t.Error("controllerHasAggro should be false after decay")
	}
}

// TestAggroDecayTaskNoEmitWhenNoAggroToFlip verifies that a non-boss monster
// without active aggro (no controller, or already passive) does NOT emit any
// event when its entries decay to zero — there's no state change to broadcast.
func TestAggroDecayTaskNoEmitWhenNoAggroToFlip(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
	// No controller -> ApplyDamage cannot flip aggro on.
	if _, err := r.ApplyDamage(ten, 1, 1, m.UniqueId(), 0); err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}

	tk, events, _ := newAggroTaskWithRecorder(t, nil)
	tk.nowFn = func() int64 { return AggroIdleThresholdMs + 1_000 }
	tk.Run()

	if len(*events) != 0 {
		t.Fatalf("expected no events, got %d (%+v)", len(*events), *events)
	}
}

// TestAggroDecayTaskBossExemption: boss monsters skip decay entirely.
func TestAggroDecayTaskBossExemption(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	bossTemplate := uint32(8800000)
	m := r.CreateMonster(ctx, ten, f, bossTemplate, 0, 0, 0, 5, 0, 1000, 50)
	if _, err := r.ControlMonster(ten, m.UniqueId(), 42); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}
	if _, err := r.ApplyDamage(ten, 1, 1, m.UniqueId(), 0); err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}

	tk, events, _ := newAggroTaskWithRecorder(t, map[uint32]bool{bossTemplate: true})
	tk.nowFn = func() int64 { return AggroIdleThresholdMs + 1_000 }
	tk.Run()

	if len(*events) != 0 {
		t.Fatalf("boss should be skipped, got %d events", len(*events))
	}
	got, _ := r.GetMonster(ten, m.UniqueId())
	if got.ControlCharacterId() != 42 {
		t.Errorf("boss controller cleared unexpectedly")
	}
	if len(got.DamageEntries()) != 1 {
		t.Errorf("boss damage entries decayed unexpectedly")
	}
}

// TestAggroDecayTaskBossCacheHitsLookupOncePerTemplate verifies the per-tick
// boss-flag cache.
func TestAggroDecayTaskBossCacheHitsLookupOncePerTemplate(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	// 3 monsters of the same template, plus 1 of a different template.
	for i := 0; i < 3; i++ {
		m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
		if _, err := r.ApplyDamage(ten, 1, 100, m.UniqueId(), 0); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	other := r.CreateMonster(ctx, ten, f, 9300019, 0, 0, 0, 5, 0, 1000, 50)
	if _, err := r.ApplyDamage(ten, 1, 100, other.UniqueId(), 0); err != nil {
		t.Fatalf("seed: %v", err)
	}

	tk, _, calls := newAggroTaskWithRecorder(t, nil)
	tk.nowFn = func() int64 { return AggroIdleThresholdMs + 1_000 }
	tk.Run()

	// Two distinct templates -> 2 lookups regardless of monster count.
	if *calls != 2 {
		t.Errorf("bossLookupFn called %d times, want 2", *calls)
	}
}

// TestAggroDecayTaskNoOpWhenAllFresh: monsters whose entries are all fresh are
// not touched and don't emit STOP_CONTROL.
func TestAggroDecayTaskNoOpWhenAllFresh(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
	if _, err := r.ControlMonster(ten, m.UniqueId(), 42); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}
	now := int64(20_000)
	if _, err := r.ApplyDamage(ten, 1, 100, m.UniqueId(), now); err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}

	tk, events, _ := newAggroTaskWithRecorder(t, nil)
	tk.nowFn = func() int64 { return now }
	tk.Run()
	if len(*events) != 0 {
		t.Fatalf("expected no events, got %d", len(*events))
	}
}

// SleepTime returns the configured interval.
func TestAggroDecayTaskSleepTime(t *testing.T) {
	tk := &MonsterAggroDecayTask{interval: 1500 * time.Millisecond}
	if tk.SleepTime() != 1500*time.Millisecond {
		t.Errorf("SleepTime=%v, want 1500ms", tk.SleepTime())
	}
}
