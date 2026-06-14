package door

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type Model struct {
	areaDoorId       uint32
	townDoorId       uint32
	ownerCharacterId uint32
	partyId          uint32
	skillId          uint32
	skillLevel       byte
	fld              field.Model
	townMapId        _map.Id
	slot             byte
	townPortalId     uint32
	areaX            int16
	areaY            int16
	townX            int16
	townY            int16
	deployTime       time.Time
	expiresAt        time.Time
}

func (m Model) AreaDoorId() uint32       { return m.areaDoorId }
func (m Model) TownDoorId() uint32       { return m.townDoorId }
func (m Model) PairId() uint32           { return m.areaDoorId }
func (m Model) OwnerCharacterId() uint32 { return m.ownerCharacterId }
func (m Model) PartyId() uint32          { return m.partyId }
func (m Model) SkillId() uint32          { return m.skillId }
func (m Model) SkillLevel() byte         { return m.skillLevel }
func (m Model) Field() field.Model       { return m.fld }
func (m Model) TownMapId() _map.Id       { return m.townMapId }
func (m Model) Slot() byte               { return m.slot }
func (m Model) TownPortalId() uint32     { return m.townPortalId }
func (m Model) AreaX() int16             { return m.areaX }
func (m Model) AreaY() int16             { return m.areaY }
func (m Model) TownX() int16             { return m.townX }
func (m Model) TownY() int16             { return m.townY }
func (m Model) DeployTime() time.Time    { return m.deployTime }
func (m Model) ExpiresAt() time.Time     { return m.expiresAt }

func (m Model) Reslot(slot byte, townPortalId uint32, townX int16, townY int16) Model {
	return Clone(m).SetSlot(slot).SetTownPortalId(townPortalId).SetTownX(townX).SetTownY(townY).Build()
}
