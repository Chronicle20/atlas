package merchant

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

type Model struct {
	id           uuid.UUID
	characterId  uint32
	shopType     byte
	state        byte
	title        string
	worldId      world.Id
	channelId    channel.Id
	mapId        uint32
	instanceId   uuid.UUID
	x            int16
	y            int16
	permitItemId uint32
	mesoBalance  uint32
	listingCount int64
	visitors     []uint32
	listings     []ListingModel
}

func (m Model) Id() uuid.UUID            { return m.id }
func (m Model) CharacterId() uint32       { return m.characterId }
func (m Model) ShopType() byte            { return m.shopType }
func (m Model) State() byte               { return m.state }
func (m Model) Title() string             { return m.title }
func (m Model) WorldId() world.Id         { return m.worldId }
func (m Model) ChannelId() channel.Id     { return m.channelId }
func (m Model) MapId() uint32             { return m.mapId }
func (m Model) InstanceId() uuid.UUID     { return m.instanceId }
func (m Model) X() int16                  { return m.x }
func (m Model) Y() int16                  { return m.y }
func (m Model) PermitItemId() uint32      { return m.permitItemId }
func (m Model) MesoBalance() uint32       { return m.mesoBalance }
func (m Model) ListingCount() int64       { return m.listingCount }
func (m Model) Visitors() []uint32        { return m.visitors }
func (m Model) Listings() []ListingModel  { return m.listings }
