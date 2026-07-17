package warp

import (
	"atlas-maps/character/location"
	"context"
	"testing"

	characterKafka "atlas-maps/kafka/message/character"
	mapsproducer "atlas-maps/kafka/producer"

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

func (c *capturingProducer) Provider() mapsproducer.Provider {
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
	p := newProcessorWithDeps(logrus.New(), ctx, lp, cp.Provider(), mt)

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
}
