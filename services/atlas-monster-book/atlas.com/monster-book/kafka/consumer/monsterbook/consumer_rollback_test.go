package monsterbook

import (
	"testing"

	"atlas-monster-book/card"
	"atlas-monster-book/collection"
	mbmsg "atlas-monster-book/kafka/message/monsterbook"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// CARD_PICKED_UP upserts a card row then recomputes the collection book level
// (class A: monster_book_cards + monster_book_collections). Failing the
// collection write must roll back the card upsert — the pair moves together.
func TestHandleCardPickedUp_RollsBackCardUpsertWhenCollectionWriteFails(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, card.Migration, collection.Migration)
	ctx := databasetest.TenantContext(uuid.New())

	databasetest.FailWritesOn(t, db, "monster_book_collections")

	l, _ := test.NewNullLogger()
	cmd := mbmsg.Command[mbmsg.CardPickedUpBody]{
		CharacterId: 1001,
		EventId:     uuid.New(),
		Type:        mbmsg.CommandTypeCardPickedUp,
		Body:        mbmsg.CardPickedUpBody{CardId: 2380000},
	}
	handleCardPickedUp(db)(l, ctx, cmd)

	var cards int64
	require.NoError(t, db.Table("monster_book_cards").Count(&cards).Error)
	require.Zero(t, cards, "card upsert must roll back with the failed collection write")
}
