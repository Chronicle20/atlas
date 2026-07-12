package character_test

import (
	"atlas-character/character"
	"atlas-character/kafka/message"
	"context"
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func outboxTestDb(t *testing.T) *gorm.DB {
	db := testDatabase(t)
	require.NoError(t, outbox.Migration(db))
	return db
}

func createTestCharacter(t *testing.T, ctx context.Context, db *gorm.DB, meso uint32) character.Model {
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("Atlas").SetLevel(1).SetExperience(0).Build()
	c, err := character.NewProcessor(testLogger(), ctx, db).Create(message.NewBuffer())(uuid.New(), input, _map.Id(0))
	require.NoError(t, err)
	if meso > 0 {
		require.NoError(t, character.NewProcessor(testLogger(), ctx, db).RequestChangeMeso(uuid.New(), c.Id(), int32(meso), 0, "SYSTEM", false))
	}
	return c
}

func outboxRowCount(t *testing.T, db *gorm.DB) int64 {
	var n int64
	require.NoError(t, db.Model(&outbox.Entity{}).Count(&n).Error)
	return n
}

func TestRequestChangeMeso_CommitEnqueuesExactlyTwoRows(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := outboxTestDb(t)
	c := createTestCharacter(t, tctx, db, 0)

	before := outboxRowCount(t, db)
	require.NoError(t, character.NewProcessor(testLogger(), tctx, db).RequestChangeMeso(uuid.New(), c.Id(), 500, 0, "SYSTEM", false))
	require.Equal(t, before+2, outboxRowCount(t, db)) // MESO_CHANGED + STAT_CHANGED

	got, err := character.NewProcessor(testLogger(), tctx, db).GetById()(c.Id())
	require.NoError(t, err)
	require.Equal(t, uint32(500), got.Meso())
}

func TestRequestChangeMeso_NotEnoughMesoEmitsNoOutboxRowsAndReturnsNil(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := outboxTestDb(t)
	c := createTestCharacter(t, tctx, db, 0)

	before := outboxRowCount(t, db)
	require.NoError(t, character.NewProcessor(testLogger(), tctx, db).RequestChangeMeso(uuid.New(), c.Id(), -100, 0, "SYSTEM", false))
	require.Equal(t, before, outboxRowCount(t, db))
}

func TestRequestChangeMeso_OverflowReturnsError(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := outboxTestDb(t)
	c := createTestCharacter(t, tctx, db, 10)

	before := outboxRowCount(t, db)
	err := character.NewProcessor(testLogger(), tctx, db).RequestChangeMeso(uuid.New(), c.Id(), 2147483647, 0, "SYSTEM", false)
	err2 := character.NewProcessor(testLogger(), tctx, db).RequestChangeMeso(uuid.New(), c.Id(), 2147483647, 0, "SYSTEM", false)
	// two max-int32 adds from a 10+2147483647 base guarantee crossing MaxUint32
	require.True(t, err != nil || err2 != nil)
	require.ErrorIs(t, func() error {
		if err != nil {
			return err
		}
		return err2
	}(), character.ErrMesoOverflow)
	_ = before
}
