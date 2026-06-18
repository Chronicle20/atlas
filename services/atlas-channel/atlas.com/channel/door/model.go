package door

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

type Model struct {
	id               string
	areaDoorId       uint32
	townDoorId       uint32
	pairId           uint32
	ownerCharacterId uint32
	partyId          uint32
	field            field.Model
	townMapId        _map.Id
	slot             byte
	townPortalId     uint32
	areaX            int16
	areaY            int16
	townX            int16
	townY            int16
	skillId          uint32
	skillLevel       byte
	expiresAt        time.Time
}

func (m Model) Id() string               { return m.id }
func (m Model) AreaDoorId() uint32       { return m.areaDoorId }
func (m Model) TownDoorId() uint32       { return m.townDoorId }
func (m Model) PairId() uint32           { return m.pairId }
func (m Model) OwnerCharacterId() uint32 { return m.ownerCharacterId }
func (m Model) PartyId() uint32          { return m.partyId }
func (m Model) Field() field.Model       { return m.field }
func (m Model) WorldId() world.Id        { return m.field.WorldId() }
func (m Model) ChannelId() channel.Id   { return m.field.ChannelId() }
func (m Model) MapId() _map.Id          { return m.field.MapId() }
func (m Model) Instance() uuid.UUID     { return m.field.Instance() }
func (m Model) TownMapId() _map.Id      { return m.townMapId }
func (m Model) Slot() byte              { return m.slot }
func (m Model) TownPortalId() uint32    { return m.townPortalId }
func (m Model) AreaX() int16            { return m.areaX }
func (m Model) AreaY() int16            { return m.areaY }
func (m Model) TownX() int16            { return m.townX }
func (m Model) TownY() int16            { return m.townY }
func (m Model) SkillId() uint32         { return m.skillId }
func (m Model) SkillLevel() byte        { return m.skillLevel }
func (m Model) ExpiresAt() time.Time    { return m.expiresAt }
