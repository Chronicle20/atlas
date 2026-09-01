package character

import (
	"atlas-buddies/buddy"
	character2 "atlas-buddies/kafka/message/character"
	listmessage "atlas-buddies/kafka/message/list"
	"atlas-buddies/list"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestMain sets every topic token env var this package's tests rely on to
// its own name, so topic.EnvProvider resolves to the same literal the
// pre-existing assertions were already written against.
func TestMain(m *testing.M) {
	_ = os.Setenv(string(listmessage.EnvStatusEventTopic), string(listmessage.EnvStatusEventTopic))
	os.Exit(m.Run())
}

func setupTestLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l, _ := test.NewNullLogger()
	return l
}

func setupTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return ten
}

func setupTestContext(t *testing.T, ten tenant.Model) context.Context {
	t.Helper()
	return tenant.WithContext(context.Background(), ten)
}

func setupTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	dbName := uuid.New().String()
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	l := setupTestLogger(t)
	database.RegisterTenantCallbacks(l, db)

	// list.Migration/buddy.Migration AutoMigrate against a uuid_generate_v4()
	// default that only postgres provides; mirror list/processor_test.go's
	// setupProcessorTestDB and create the tables directly for sqlite.
	if err = db.Exec(`
		CREATE TABLE lists (
			tenant_id TEXT NOT NULL,
			id TEXT PRIMARY KEY,
			character_id INTEGER NOT NULL,
			capacity INTEGER NOT NULL
		)
	`).Error; err != nil {
		t.Fatalf("Failed to create lists table: %v", err)
	}
	if err = db.Exec(`
		CREATE TABLE buddies (
			character_id INTEGER PRIMARY KEY,
			list_id TEXT NOT NULL,
			tenant_id TEXT,
			"group" TEXT NOT NULL,
			character_name TEXT NOT NULL,
			channel_id INTEGER NOT NULL DEFAULT -1,
			in_shop BOOLEAN NOT NULL DEFAULT false,
			pending BOOLEAN NOT NULL DEFAULT false
		)
	`).Error; err != nil {
		t.Fatalf("Failed to create buddies table: %v", err)
	}

	// UpdateBuddyNameAndEmit routes its BUDDY_UPDATED events through the
	// outbox (list/processor.go:610-645's pattern); outbox.Entity has no
	// postgres-specific defaults so AutoMigrate works against sqlite.
	if err = outbox.Migration(db); err != nil {
		t.Fatalf("Failed to migrate outbox: %v", err)
	}

	return db
}

// seedBuddyList creates ownerId's own buddy list entity (capacity 20) plus
// one buddy row on it naming buddyId as buddyName. updateBuddyName matches
// on the *buddy's* character id — the row lives on the owner's list but is
// keyed by the buddy it names — so seeding correctly here is what makes the
// "wrong owner" bug reproducible in the tests below.
func seedBuddyList(t *testing.T, db *gorm.DB, ten tenant.Model, ownerId uint32, buddyId uint32, buddyName string) {
	t.Helper()
	le := list.Entity{
		TenantId:    ten.Id(),
		Id:          uuid.New(),
		CharacterId: ownerId,
		Capacity:    20,
	}
	if err := db.Create(&le).Error; err != nil {
		t.Fatalf("Failed to seed list for owner [%d]: %v", ownerId, err)
	}

	be := buddy.Entity{
		CharacterId:   buddyId,
		ListId:        le.Id,
		TenantId:      ten.Id(),
		Group:         "Default Group",
		CharacterName: buddyName,
		ChannelId:     -1,
	}
	if err := db.Create(&be).Error; err != nil {
		t.Fatalf("Failed to seed buddy [%d] on owner [%d] list: %v", buddyId, ownerId, err)
	}
}

func buddyName(t *testing.T, db *gorm.DB, ten tenant.Model, ownerId uint32, buddyId uint32) string {
	t.Helper()
	var le list.Entity
	if err := db.Where("tenant_id = ? AND character_id = ?", ten.Id(), ownerId).First(&le).Error; err != nil {
		t.Fatalf("Failed to load list for owner [%d]: %v", ownerId, err)
	}
	var be buddy.Entity
	if err := db.Where("list_id = ? AND character_id = ?", le.Id, buddyId).First(&be).Error; err != nil {
		t.Fatalf("Failed to load buddy [%d] on owner [%d] list: %v", buddyId, ownerId, err)
	}
	return be.CharacterName
}

// buddyUpdatedOutboxCount counts the BUDDY_UPDATED status events enqueued
// (via outbox.EmitProvider, see list/processor.go:723-724) for ownerId on
// the buddy list status event topic. Filtering by decoded event type (rather
// than just row count on the topic) keeps this robust against other status
// event types sharing the same topic.
func buddyUpdatedOutboxCount(t *testing.T, db *gorm.DB, ten tenant.Model, ownerId uint32) int {
	t.Helper()
	var rows []outbox.Entity
	if err := db.Where("topic = ?", listmessage.EnvStatusEventTopic).Find(&rows).Error; err != nil {
		t.Fatalf("Failed to query outbox entries: %v", err)
	}
	count := 0
	for _, r := range rows {
		var ev listmessage.StatusEvent[listmessage.BuddyUpdatedStatusEventBody]
		if err := json.Unmarshal(r.MessageValue, &ev); err != nil {
			continue
		}
		if ev.Type == listmessage.StatusEventTypeBuddyUpdated && uint32(ev.CharacterId) == ownerId {
			count++
		}
	}
	return count
}

func nameChangedEvent(characterId uint32, oldName string, newName string) character2.StatusEvent[character2.NameChangedStatusEventBody] {
	return character2.StatusEvent[character2.NameChangedStatusEventBody]{
		TransactionId: uuid.New(),
		WorldId:       world.Id(0),
		CharacterId:   characterId,
		Type:          character2.StatusEventTypeNameChanged,
		Body:          character2.NameChangedStatusEventBody{OldName: oldName, NewName: newName},
	}
}

// TestNameChangedUpdatesEveryOwnersCopyOfTheBuddyName is the wrong-column
// regression test called for by the task-227 brief. The plan's original
// draft seeded two owners both holding character 1 as a buddy, but
// buddy.Entity's sole primary key is character_id (buddy/entity.go:24), a
// pre-existing schema defect this task does not fix, so that scenario cannot
// be seeded. Instead: owner 10 holds character 1 as a buddy, and character 1
// separately owns its own list naming character 99 as a buddy. A handler
// that matched on the owner (characterId) instead of the buddy (targetId)
// would instead update character 1's OWN list — this test's second
// assertion catches exactly that.
func TestNameChangedUpdatesEveryOwnersCopyOfTheBuddyName(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	seedBuddyList(t, db, ten, 10, 1, "Yankee")
	seedBuddyList(t, db, ten, 1, 99, "Whoever")

	handleStatusEventNameChanged(db)(l, ctx, nameChangedEvent(1, "Yankee", "Zulu"))

	require.Equal(t, "Zulu", buddyName(t, db, ten, 10, 1))
	require.Equal(t, "Whoever", buddyName(t, db, ten, 1, 99))
}

// At-least-once delivery: a redelivered event must be a harmless no-op, not
// a second BUDDY_UPDATED emit. updateBuddyName's "already matches" check is
// what gates this.
func TestNameChangedIsIdempotent(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	seedBuddyList(t, db, ten, 10, 1, "Yankee")

	ev := nameChangedEvent(1, "Yankee", "Zulu")
	handleStatusEventNameChanged(db)(l, ctx, ev)
	require.Equal(t, 1, buddyUpdatedOutboxCount(t, db, ten, 10), "first delivery must emit exactly one BUDDY_UPDATED")

	handleStatusEventNameChanged(db)(l, ctx, ev)
	require.Equal(t, 1, buddyUpdatedOutboxCount(t, db, ten, 10), "redelivery of the identical event must not emit a second BUDDY_UPDATED")

	require.Equal(t, "Zulu", buddyName(t, db, ten, 10, 1))
}

// A character nobody has as a buddy must be a silent no-op: no error, no
// row created.
func TestNameChangedForANonBuddyIsANoOp(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	handleStatusEventNameChanged(db)(l, ctx, nameChangedEvent(99, "A", "B"))

	var count int64
	if err := db.Model(&buddy.Entity{}).Count(&count).Error; err != nil {
		t.Fatalf("Failed to count buddies: %v", err)
	}
	require.EqualValues(t, 0, count)
}
