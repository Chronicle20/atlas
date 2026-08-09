package warp

import (
	"atlas-maps/character/location"
	"atlas-maps/data/map/info"
	"context"
	"errors"
	"testing"

	characterKafka "atlas-maps/kafka/message/character"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	kafkaproducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// capturingProducer records every emitted message by topic.
type capturingProducer struct {
	messages map[string][]kafka.Message
}

func newCapturingProducer() *capturingProducer {
	return &capturingProducer{messages: make(map[string][]kafka.Message)}
}

func (c *capturingProducer) Provider() kafkaproducer.Provider {
	return func(token string) kafkaproducer.MessageProducer {
		return func(p model.Provider[[]kafka.Message]) error {
			ms, err := p()
			if err != nil {
				return err
			}
			c.messages[token] = append(c.messages[token], ms...)
			return nil
		}
	}
}

// noopTransitioner satisfies mapTransitioner without external calls.
type noopTransitioner struct{ calls int }

func (n *noopTransitioner) TransitionMapAndEmit(_ uuid.UUID, _ field.Model, _ uint32, _ field.Model) error {
	n.calls++
	return nil
}

// recordingTimer satisfies timer.Processor and records the map-time-limit
// hooks warp.ChangeMap fires (task-050, relocated off the MAP_CHANGED
// consumer by issue #1192).
type recordingTimer struct {
	cancels      int
	registered   []_map.Id
	forcedReturn []_map.Id
	seconds      []uint32
}

func (r *recordingTimer) Register(_ uuid.UUID, _ uint32, f field.Model, forcedReturnMapId _map.Id, seconds uint32) error {
	r.registered = append(r.registered, f.MapId())
	r.forcedReturn = append(r.forcedReturn, forcedReturnMapId)
	r.seconds = append(r.seconds, seconds)
	return nil
}

func (r *recordingTimer) CancelIfTracked(_ uint32) bool {
	r.cancels++
	return true
}

func (r *recordingTimer) ForceReturnIfTracked(_ uint32) bool { return false }

// stubMapInfo satisfies info.Processor from a fixed table; a map absent from
// the table returns an error, mirroring an atlas-data lookup failure.
type stubMapInfo struct{ byId map[_map.Id]info.Model }

func (s stubMapInfo) GetById(mapId _map.Id) (info.Model, error) {
	m, ok := s.byId[mapId]
	if !ok {
		return info.Model{}, errors.New("map info unavailable")
	}
	return m, nil
}

func newCtxTenant(t *testing.T) context.Context {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tn)
}

func newLocationDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := location.Migration(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func world0() world.Id     { return 0 }
func channel1() channel.Id { return 1 }

func TestChangeMap_PersistsAndEmitsMapChanged(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newLocationDB(t)
	lp := location.NewProcessor(logrus.New(), ctx, db)

	// Seed an existing location row (the "old" side).
	start := field.NewBuilder(world0(), channel1(), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	if _, err := lp.Set(12345, start); err != nil {
		t.Fatalf("seed Set: %v", err)
	}

	cp := newCapturingProducer()
	mt := &noopTransitioner{}
	tm := &recordingTimer{}
	p := newProcessorWithDeps(logrus.New(), ctx, lp, cp.Provider(), mt, tm, stubMapInfo{})

	dest := field.NewBuilder(world0(), channel1(), _map.Id(104000000)).SetInstance(uuid.Nil).Build()
	if err := p.ChangeMap(uuid.New(), 12345, world0(), dest, 0, false, 0, 0); err != nil {
		t.Fatalf("ChangeMap: %v", err)
	}

	got, err := lp.GetById(12345)
	if err != nil {
		t.Fatalf("GetById after warp: %v", err)
	}
	if got.MapId() != _map.Id(104000000) {
		t.Fatalf("persisted MapId = %d, want 104000000", got.MapId())
	}

	msgs := cp.messages[characterKafka.EnvEventTopicCharacterStatus]
	if len(msgs) != 1 {
		t.Fatalf("emitted %d status messages, want 1", len(msgs))
	}
	if mt.calls != 1 {
		t.Fatalf("TransitionMapAndEmit called %d times, want 1", mt.calls)
	}
	if tm.cancels != 1 {
		t.Fatalf("CancelIfTracked called %d times, want 1", tm.cancels)
	}
	if len(tm.registered) != 0 {
		t.Fatalf("registered timers %v, want none for a map with no info", tm.registered)
	}
}

// The map-time-limit hooks used to hang off atlas-maps' own MAP_CHANGED
// consumer, which was removed with issue #1192. They must still fire on the
// warp path — ChangeMap is the sole emitter of MAP_CHANGED.
func TestChangeMap_RegistersTimerForTimeLimitedDestination(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newLocationDB(t)
	lp := location.NewProcessor(logrus.New(), ctx, db)

	dest := field.NewBuilder(world0(), channel1(), _map.Id(280030000)).SetInstance(uuid.Nil).Build()
	cp := newCapturingProducer()
	tm := &recordingTimer{}
	ip := stubMapInfo{byId: map[_map.Id]info.Model{
		_map.Id(280030000): info.NewBuilder().
			SetId(_map.Id(280030000)).
			SetTimeLimit(600).
			SetForcedReturnMapId(_map.Id(240000110)).
			Build(),
	}}
	p := newProcessorWithDeps(logrus.New(), ctx, lp, cp.Provider(), &noopTransitioner{}, tm, ip)

	if err := p.ChangeMap(uuid.New(), 12345, world0(), dest, 0, false, 0, 0); err != nil {
		t.Fatalf("ChangeMap: %v", err)
	}

	if tm.cancels != 1 {
		t.Fatalf("CancelIfTracked called %d times, want 1", tm.cancels)
	}
	if len(tm.registered) != 1 || tm.registered[0] != _map.Id(280030000) {
		t.Fatalf("registered = %v, want [280030000]", tm.registered)
	}
	if tm.forcedReturn[0] != _map.Id(240000110) {
		t.Fatalf("forcedReturn = %d, want 240000110", tm.forcedReturn[0])
	}
	if tm.seconds[0] != 600 {
		t.Fatalf("seconds = %d, want 600", tm.seconds[0])
	}
}

func TestChangeMap_NoTimerForUnlimitedDestination(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newLocationDB(t)
	lp := location.NewProcessor(logrus.New(), ctx, db)

	dest := field.NewBuilder(world0(), channel1(), _map.Id(104000000)).SetInstance(uuid.Nil).Build()
	cp := newCapturingProducer()
	tm := &recordingTimer{}
	ip := stubMapInfo{byId: map[_map.Id]info.Model{
		_map.Id(104000000): info.NewBuilder().SetId(_map.Id(104000000)).Build(),
	}}
	p := newProcessorWithDeps(logrus.New(), ctx, lp, cp.Provider(), &noopTransitioner{}, tm, ip)

	if err := p.ChangeMap(uuid.New(), 12345, world0(), dest, 0, false, 0, 0); err != nil {
		t.Fatalf("ChangeMap: %v", err)
	}

	if tm.cancels != 1 {
		t.Fatalf("CancelIfTracked called %d times, want 1", tm.cancels)
	}
	if len(tm.registered) != 0 {
		t.Fatalf("registered = %v, want none for an unlimited map", tm.registered)
	}
}
