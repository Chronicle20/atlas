package monsterbook

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/monster"
)

// Model is the character's monster-book view: the collection summary (cover,
// book level, normal/special/total counts, exp bonus) plus the owned-card list.
// The zero value is an empty book — what an undecorated character carries.
type Model struct {
	collection Collection
	cards      []Card
}

// NewModel composes a monster-book model from a collection summary and the
// owned-card list.
func NewModel(collection Collection, cards []Card) Model {
	return Model{collection: collection, cards: cards}
}

func (m Model) CoverCardId() item.Id       { return m.collection.CoverCardId() }
func (m Model) CoverMonsterId() monster.Id { return m.collection.CoverMonsterId() }
func (m Model) Level() uint16              { return m.collection.BookLevel() }
func (m Model) NormalCount() uint16        { return m.collection.NormalCount() }
func (m Model) SpecialCount() uint16       { return m.collection.SpecialCount() }
func (m Model) TotalUniqueCards() uint16   { return m.collection.TotalUniqueCards() }
func (m Model) ExpBonusPercent() uint16    { return m.collection.ExpBonusPercent() }
func (m Model) Cards() []Card              { return m.cards }
