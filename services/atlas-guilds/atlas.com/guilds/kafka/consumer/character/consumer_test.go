package character

import (
	"atlas-guilds/guild/member"
	character2 "atlas-guilds/kafka/message/character"
	"context"
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
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

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

	if err = member.Migration(db); err != nil {
		t.Fatalf("Failed to migrate member: %v", err)
	}

	return db
}

func seedGuildMember(t *testing.T, db *gorm.DB, ten tenant.Model, guildId uint32, characterId uint32, name string) {
	t.Helper()
	e := member.Entity{
		TenantId:    ten.Id(),
		GuildId:     guildId,
		CharacterId: characterId,
		Name:        name,
		Level:       1,
		Title:       5,
	}
	if err := db.Create(&e).Error; err != nil {
		t.Fatalf("Failed to seed guild member: %v", err)
	}
}

func guildMemberName(t *testing.T, db *gorm.DB, ten tenant.Model, characterId uint32) string {
	t.Helper()
	var e member.Entity
	if err := db.Where("tenant_id = ? AND character_id = ?", ten.Id(), characterId).First(&e).Error; err != nil {
		t.Fatalf("Failed to load guild member: %v", err)
	}
	return e.Name
}

func guildMemberRowCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&member.Entity{}).Count(&count).Error; err != nil {
		t.Fatalf("Failed to count guild members: %v", err)
	}
	return count
}

func TestNameChangedUpdatesTheGuildRosterName(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)
	seedGuildMember(t, db, ten, 5, 1, "Yankee")

	handleCharacterNameChanged(db)(l, ctx, character2.StatusEvent[character2.StatusEventNameChangedBody]{
		TransactionId: uuid.New(),
		WorldId:       world.Id(0),
		CharacterId:   1,
		Type:          character2.EventCharacterStatusTypeNameChanged,
		Body:          character2.StatusEventNameChangedBody{OldName: "Yankee", NewName: "Zulu"},
	})

	require.Equal(t, "Zulu", guildMemberName(t, db, ten, 1))
}

// At-least-once delivery: a redelivered event must be a harmless no-op.
func TestNameChangedIsIdempotent(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)
	seedGuildMember(t, db, ten, 5, 1, "Yankee")

	ev := character2.StatusEvent[character2.StatusEventNameChangedBody]{
		TransactionId: uuid.New(),
		WorldId:       world.Id(0),
		CharacterId:   1,
		Type:          character2.EventCharacterStatusTypeNameChanged,
		Body:          character2.StatusEventNameChangedBody{OldName: "Yankee", NewName: "Zulu"},
	}
	handleCharacterNameChanged(db)(l, ctx, ev)
	handleCharacterNameChanged(db)(l, ctx, ev)

	require.Equal(t, "Zulu", guildMemberName(t, db, ten, 1))
}

// A character with no guild membership must not error or create a row.
func TestNameChangedForANonMemberIsANoOp(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupTestDatabase(t)

	handleCharacterNameChanged(db)(l, ctx, character2.StatusEvent[character2.StatusEventNameChangedBody]{
		TransactionId: uuid.New(),
		WorldId:       world.Id(0),
		CharacterId:   99,
		Type:          character2.EventCharacterStatusTypeNameChanged,
		Body:          character2.StatusEventNameChangedBody{OldName: "A", NewName: "B"},
	})

	require.EqualValues(t, 0, guildMemberRowCount(t, db))
}
