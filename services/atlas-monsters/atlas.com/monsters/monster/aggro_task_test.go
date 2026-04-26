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
	Topic   string
	Type    string
	ActorId uint32
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
						ActorId uint32 `json:"actorId"`
					} `json:"body"`
				}
				if err := json.Unmarshal(m.Value, &env); err != nil {
					t.Fatalf("decode: %v", err)
				}
				events = append(events, recordedEmit{Topic: topic, Type: env.Type, ActorId: env.Body.ActorId})
			}
			return nil
		},
	}
	return tk, &events, &bossCalls
}

// TestAggroDecayTaskFullClearEmitsStopControl seeds a non-boss monster with a
// tiny entry and a controller, fast-forwards the wall-clock past the idle
// threshold, runs Run(), and asserts STOP_CONTROL is emitted with the previous
// controller id.
func TestAggroDecayTaskFullClearEmitsStopControl(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
	if _, err := r.ControlMonster(ten, m.UniqueId(), 42); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}
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
	if (*events)[0].Type != EventMonsterStatusStopControl {
		t.Errorf("type=%s, want STOP_CONTROL", (*events)[0].Type)
	}
	if (*events)[0].ActorId != 42 {
		t.Errorf("ActorId=%d, want 42", (*events)[0].ActorId)
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
