package mount_test

import (
	"atlas-mounts/kafka/message"
	mountmsg "atlas-mounts/kafka/message/mount"
	"atlas-mounts/mount"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

func testContext(t *testing.T) context.Context {
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), tn)
}

func testDatabase(t *testing.T) *gorm.DB {
	l := testLogger()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	database.RegisterTenantCallbacks(l, db)

	if err = mount.Migration(db); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

func statusEvents(t *testing.T, mb *message.Buffer) []kafka.Message {
	ke := mb.GetAll()
	se, ok := ke[mountmsg.EnvStatusEventTopic]
	if !ok {
		t.Fatalf("Failed to get events from topic: %s", mountmsg.EnvStatusEventTopic)
	}
	return se
}

func TestProcessor_GetByCharacterId_DefaultOnFirstRead(t *testing.T) {
	db := testDatabase(t)
	p := mount.NewProcessor(testLogger(), testContext(t), db)

	characterId := uint32(7000000)
	m, err := p.GetByCharacterId(characterId)
	if err != nil {
		t.Fatalf("Failed to get/create mount: %v", err)
	}
	if m.Id() == uuid.Nil {
		t.Fatalf("Default mount should have a generated id")
	}
	if m.CharacterId() != characterId {
		t.Fatalf("CharacterId mismatch: got %d", m.CharacterId())
	}
	if m.Level() != 1 {
		t.Fatalf("Default mount level should be 1, got %d", m.Level())
	}
	if m.Exp() != 0 {
		t.Fatalf("Default mount exp should be 0, got %d", m.Exp())
	}
	if m.Tiredness() != 0 {
		t.Fatalf("Default mount tiredness should be 0, got %d", m.Tiredness())
	}

	// A second read must return the same persisted row (no duplicate).
	m2, err := p.GetByCharacterId(characterId)
	if err != nil {
		t.Fatalf("Failed to re-read mount: %v", err)
	}
	if m2.Id() != m.Id() {
		t.Fatalf("Second read should return the same row id; got %s vs %s", m2.Id(), m.Id())
	}
}

func TestProcessor_GetByCharacterId_TenantScoped(t *testing.T) {
	db := testDatabase(t)
	characterId := uint32(7000001)

	ctxA := testContext(t)
	ctxB := testContext(t)

	pa := mount.NewProcessor(testLogger(), ctxA, db)
	pb := mount.NewProcessor(testLogger(), ctxB, db)

	ma, err := pa.GetByCharacterId(characterId)
	if err != nil {
		t.Fatalf("tenant A: failed to create mount: %v", err)
	}
	mb, err := pb.GetByCharacterId(characterId)
	if err != nil {
		t.Fatalf("tenant B: failed to create mount: %v", err)
	}

	if ma.Id() == mb.Id() {
		t.Fatalf("Two tenants with the same characterId must get independent rows")
	}
	if ma.TenantId() == mb.TenantId() {
		t.Fatalf("Tenant ids should differ between the two fixtures")
	}

	tA := tenant.MustFromContext(ctxA)
	tB := tenant.MustFromContext(ctxB)
	if ma.TenantId() != tA.Id() {
		t.Fatalf("tenant A row has wrong tenant id")
	}
	if mb.TenantId() != tB.Id() {
		t.Fatalf("tenant B row has wrong tenant id")
	}
}

func TestProcessor_ApplyTick(t *testing.T) {
	db := testDatabase(t)
	p := mount.NewProcessor(testLogger(), testContext(t), db)

	characterId := uint32(7000002)
	worldId := world.Id(0)

	mb := message.NewBuffer()
	if err := p.ApplyTick(mb)(worldId, characterId); err != nil {
		t.Fatalf("Failed to apply tick: %v", err)
	}

	se := statusEvents(t, mb)
	if len(se) != 1 {
		t.Fatalf("Expected 1 TICK event, got %d", len(se))
	}

	o, err := p.GetByCharacterId(characterId)
	if err != nil {
		t.Fatalf("Failed to read mount: %v", err)
	}
	if o.Tiredness() != 1 {
		t.Fatalf("Tiredness should be 1 after one tick, got %d", o.Tiredness())
	}
	if o.LastTirednessTickAt() == nil {
		t.Fatalf("LastTirednessTickAt should be set after a tick")
	}
}

func TestProcessor_ApplyFeedAndEmit(t *testing.T) {
	db := testDatabase(t)
	p := mount.NewProcessor(testLogger(), testContext(t), db)

	characterId := uint32(7000003)
	worldId := world.Id(0)

	// Seed some tiredness so the feed has something to heal.
	mb0 := message.NewBuffer()
	for i := 0; i < 10; i++ {
		if err := p.ApplyTick(mb0)(worldId, characterId); err != nil {
			t.Fatalf("Failed to seed tiredness: %v", err)
		}
	}
	before, err := p.GetByCharacterId(characterId)
	if err != nil {
		t.Fatalf("Failed to read seeded mount: %v", err)
	}
	if before.Tiredness() != 10 {
		t.Fatalf("Expected seeded tiredness 10, got %d", before.Tiredness())
	}

	healMax := 30
	// Independently compute the expected result via the pure feed math.
	expected := mount.ApplyFeed(mount.FeedInput{
		Level:     before.Level(),
		Exp:       before.Exp(),
		Tiredness: before.Tiredness(),
		HealMax:   healMax,
	})

	mb := message.NewBuffer()
	if err = p.ApplyFeedAndEmit(mb)(worldId, characterId, healMax); err != nil {
		t.Fatalf("Failed to apply feed: %v", err)
	}

	se := statusEvents(t, mb)
	if len(se) != 1 {
		t.Fatalf("Expected 1 FEED event, got %d", len(se))
	}

	o, err := p.GetByCharacterId(characterId)
	if err != nil {
		t.Fatalf("Failed to read mount after feed: %v", err)
	}
	if o.Level() != expected.Level {
		t.Fatalf("Level mismatch after feed: got %d want %d", o.Level(), expected.Level)
	}
	if o.Exp() != expected.Exp {
		t.Fatalf("Exp mismatch after feed: got %d want %d", o.Exp(), expected.Exp)
	}
	if o.Tiredness() != expected.Tiredness {
		t.Fatalf("Tiredness mismatch after feed: got %d want %d", o.Tiredness(), expected.Tiredness)
	}
}

func TestProcessor_EmitSet(t *testing.T) {
	db := testDatabase(t)
	p := mount.NewProcessor(testLogger(), testContext(t), db)

	characterId := uint32(7000004)
	worldId := world.Id(0)

	mb := message.NewBuffer()
	if err := p.EmitSet(mb)(worldId, characterId); err != nil {
		t.Fatalf("Failed to emit set: %v", err)
	}

	se := statusEvents(t, mb)
	if len(se) != 1 {
		t.Fatalf("Expected 1 SET event, got %d", len(se))
	}

	// EmitSet must default-create the row without mutating progression.
	o, err := p.GetByCharacterId(characterId)
	if err != nil {
		t.Fatalf("Failed to read mount after set: %v", err)
	}
	if o.Level() != 1 || o.Exp() != 0 || o.Tiredness() != 0 {
		t.Fatalf("EmitSet should not change progression; got level=%d exp=%d tiredness=%d", o.Level(), o.Exp(), o.Tiredness())
	}
}
