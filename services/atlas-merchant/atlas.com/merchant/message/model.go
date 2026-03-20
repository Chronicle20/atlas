package message

import (
	"time"

	"github.com/google/uuid"
)

type Model struct {
	id          uuid.UUID
	shopId      uuid.UUID
	characterId uint32
	content     string
	sentAt      time.Time
}

func (m Model) Id() uuid.UUID      { return m.id }
func (m Model) ShopId() uuid.UUID  { return m.shopId }
func (m Model) CharacterId() uint32 { return m.characterId }
func (m Model) Content() string     { return m.content }
func (m Model) SentAt() time.Time   { return m.sentAt }

func Make(e Entity) (Model, error) {
	return Model{
		id:          e.Id,
		shopId:      e.ShopId,
		characterId: e.CharacterId,
		content:     e.Content,
		sentAt:      e.SentAt,
	}, nil
}
