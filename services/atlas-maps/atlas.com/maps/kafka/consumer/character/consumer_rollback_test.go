package character

import (
	"testing"
	"time"

	"atlas-maps/character/location"
	characterKafka "atlas-maps/kafka/message/character"
	"atlas-maps/visit"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// Character-deleted deletes character_map_visits then character_locations
// (class A: two tables). Failing the locations delete must roll back the
// visits delete — the pair moves together.
func TestHandleStatusEventDeleted_RollsBackVisitDeleteWhenLocationDeleteFails(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, visit.MigrateTable, location.Migration)
	tid := uuid.New()
	ctx := databasetest.TenantContext(tid)
	l, _ := test.NewNullLogger()

	require.NoError(t, db.Create(&visit.Entity{ID: uuid.New(), TenantId: tid, CharacterID: 7001, MapID: 100000000, FirstVisitedAt: time.Now()}).Error)

	databasetest.FailWritesOn(t, db, "character_locations", databasetest.WriteDelete)

	handleStatusEventDeletedFunc(l, db)(l, ctx, characterKafka.StatusEvent[characterKafka.StatusEventDeletedBody]{
		CharacterId: 7001,
		Type:        characterKafka.EventCharacterStatusTypeDeleted,
	})

	var visits int64
	require.NoError(t, db.Table("character_map_visits").Count(&visits).Error)
	require.EqualValues(t, 1, visits, "the visit delete must roll back with the failed location delete")
}
