package record

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// setupTestDB mirrors services/atlas-buddies/atlas.com/buddies/list/administrator_test.go:
// a fresh sqlite-in-memory db with the game_records table created directly
// (Entity's `default:uuid_generate_v4()` gorm tag is PostgreSQL-only and
// can't run through AutoMigrate on sqlite), plus tenant callbacks registered
// for parity with the production Connect() path.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	database.RegisterTenantCallbacks(l, db)

	err = db.Exec(`
		CREATE TABLE game_records (
			tenant_id TEXT NOT NULL,
			id TEXT PRIMARY KEY,
			character_id INTEGER NOT NULL,
			game_type TEXT NOT NULL,
			wins INTEGER NOT NULL DEFAULT 0,
			ties INTEGER NOT NULL DEFAULT 0,
			losses INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			UNIQUE(tenant_id, character_id, game_type)
		)
	`).Error
	require.NoError(t, err)

	return db
}

// setupTenantCtx builds a fresh tenant and a context carrying it, mirroring the
// production consumer/REST path where the tenant lives in context and the
// tenant callbacks scope every query from it (DOM-11 context-driven tenancy).
func setupTenantCtx(t *testing.T) (tenant.Model, context.Context) {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return ten, tenant.WithContext(context.Background(), ten)
}

func TestApplyResult_OwnerWin(t *testing.T) {
	db := setupTestDB(t)
	_, ctx := setupTenantCtx(t)
	ownerId := uint32(100)
	visitorId := uint32(200)

	err := ApplyResult(db.WithContext(ctx), GameTypeOmok, ownerId, visitorId, 0, false)
	require.NoError(t, err)

	owner, err := GetOrZero(ctx, db.WithContext(ctx), ownerId, GameTypeOmok)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), owner.Wins())
	assert.Equal(t, uint32(0), owner.Ties())
	assert.Equal(t, uint32(0), owner.Losses())

	visitor, err := GetOrZero(ctx, db.WithContext(ctx), visitorId, GameTypeOmok)
	require.NoError(t, err)
	assert.Equal(t, uint32(0), visitor.Wins())
	assert.Equal(t, uint32(0), visitor.Ties())
	assert.Equal(t, uint32(1), visitor.Losses())
}

func TestApplyResult_VisitorWin(t *testing.T) {
	db := setupTestDB(t)
	_, ctx := setupTenantCtx(t)
	ownerId := uint32(100)
	visitorId := uint32(200)

	err := ApplyResult(db.WithContext(ctx), GameTypeOmok, ownerId, visitorId, 1, false)
	require.NoError(t, err)

	owner, err := GetOrZero(ctx, db.WithContext(ctx), ownerId, GameTypeOmok)
	require.NoError(t, err)
	assert.Equal(t, uint32(0), owner.Wins())
	assert.Equal(t, uint32(1), owner.Losses())

	visitor, err := GetOrZero(ctx, db.WithContext(ctx), visitorId, GameTypeOmok)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), visitor.Wins())
	assert.Equal(t, uint32(0), visitor.Losses())
}

func TestApplyResult_Tie(t *testing.T) {
	db := setupTestDB(t)
	_, ctx := setupTenantCtx(t)
	ownerId := uint32(100)
	visitorId := uint32(200)

	// winnerSlot is meaningless when tie is true; passing a non-zero value
	// makes sure tie really does override it rather than being ignored.
	err := ApplyResult(db.WithContext(ctx), GameTypeMatchCards, ownerId, visitorId, 1, true)
	require.NoError(t, err)

	owner, err := GetOrZero(ctx, db.WithContext(ctx), ownerId, GameTypeMatchCards)
	require.NoError(t, err)
	assert.Equal(t, uint32(0), owner.Wins())
	assert.Equal(t, uint32(1), owner.Ties())
	assert.Equal(t, uint32(0), owner.Losses())

	visitor, err := GetOrZero(ctx, db.WithContext(ctx), visitorId, GameTypeMatchCards)
	require.NoError(t, err)
	assert.Equal(t, uint32(0), visitor.Wins())
	assert.Equal(t, uint32(1), visitor.Ties())
	assert.Equal(t, uint32(0), visitor.Losses())
}

func TestApplyResult_CreatesMissingRows(t *testing.T) {
	db := setupTestDB(t)
	ten, ctx := setupTenantCtx(t)
	ownerId := uint32(300)
	visitorId := uint32(400)

	var countBefore int64
	require.NoError(t, db.Model(&Entity{}).Where("tenant_id = ?", ten.Id()).Count(&countBefore).Error)
	require.Equal(t, int64(0), countBefore, "neither row should exist before the first game")

	err := ApplyResult(db.WithContext(ctx), GameTypeOmok, ownerId, visitorId, 0, false)
	require.NoError(t, err)

	var countAfter int64
	require.NoError(t, db.Model(&Entity{}).Where("tenant_id = ?", ten.Id()).Count(&countAfter).Error)
	assert.Equal(t, int64(2), countAfter, "ApplyResult must create both the owner's and visitor's row with the context tenant injected")
}

// TestApplyResult_Atomic forces the visitor row's UPDATE to fail (a sqlite
// trigger stands in for "inject an error on the second write" per the task
// brief) and asserts that the owner's already-applied increment did not
// persist — proving the two-row upsert really runs inside one transaction
// and not as two independent writes. This guards against silently
// regressing to database.ExecuteTransaction (a confirmed no-op, see
// bug_execute_transaction_noop.md) which would let the owner's write commit
// even when the visitor's write fails.
func TestApplyResult_Atomic(t *testing.T) {
	db := setupTestDB(t)
	ten, ctx := setupTenantCtx(t)
	ownerId := uint32(500)
	visitorId := uint32(600)

	// Pre-seed the owner's row so a persisted increment is observable
	// (getOrCreate would otherwise create it fresh at Wins=0 either way).
	require.NoError(t, db.Create(&Entity{
		Id:          uuid.New(),
		TenantId:    ten.Id(),
		CharacterId: ownerId,
		GameType:    string(GameTypeOmok),
	}).Error)

	// Force any UPDATE touching the visitor's character_id to fail. getOrCreate's
	// INSERT of the visitor row is unaffected, so this fires on the visitor's
	// tx.Save (an UPDATE, since the row now has a primary key) inside ApplyResult.
	require.NoError(t, db.Exec(fmt.Sprintf(`
		CREATE TRIGGER fail_visitor_update BEFORE UPDATE ON game_records
		WHEN NEW.character_id = %d
		BEGIN SELECT RAISE(ABORT, 'forced failure for atomicity test'); END;
	`, visitorId)).Error)

	err := ApplyResult(db.WithContext(ctx), GameTypeOmok, ownerId, visitorId, 0, false)
	require.Error(t, err)

	owner, err := GetOrZero(ctx, db.WithContext(ctx), ownerId, GameTypeOmok)
	require.NoError(t, err)
	assert.Equal(t, uint32(0), owner.Wins(), "owner's increment must not persist when the transaction rolls back")

	var visitorCount int64
	require.NoError(t, db.Unscoped().Model(&Entity{}).Where("tenant_id = ? AND character_id = ?", ten.Id(), visitorId).Count(&visitorCount).Error)
	assert.Equal(t, int64(0), visitorCount, "visitor row's insert must roll back along with the failed update")
}
