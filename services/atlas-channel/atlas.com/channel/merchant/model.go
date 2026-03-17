package merchant

import "github.com/google/uuid"

type Model struct {
	id           uuid.UUID
	characterId  uint32
	shopType     byte
	title        string
	x            int16
	y            int16
	permitItemId uint32
	listingCount int64
	visitors     []uint32
}

func (m Model) Id() uuid.UUID       { return m.id }
func (m Model) CharacterId() uint32  { return m.characterId }
func (m Model) ShopType() byte       { return m.shopType }
func (m Model) Title() string        { return m.title }
func (m Model) X() int16             { return m.x }
func (m Model) Y() int16             { return m.y }
func (m Model) PermitItemId() uint32 { return m.permitItemId }
func (m Model) ListingCount() int64  { return m.listingCount }
func (m Model) Visitors() []uint32   { return m.visitors }
