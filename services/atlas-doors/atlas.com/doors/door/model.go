package door

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

type Model struct {
	areaDoorId       uint32
	townDoorId       uint32
	ownerCharacterId character.Id
	partyId          uint32
	skillId          skill.Id
	skillLevel       byte
	fld              field.Model
	townMapId        _map.Id
	slot             byte
	townPortalId     uint32
	areaX            point.X
	areaY            point.Y
	townX            point.X
	townY            point.Y
	deployTime       time.Time
	expiresAt        time.Time
}

func (m Model) AreaDoorId() uint32             { return m.areaDoorId }
func (m Model) TownDoorId() uint32             { return m.townDoorId }
func (m Model) PairId() uint32                 { return m.areaDoorId }
func (m Model) OwnerCharacterId() character.Id { return m.ownerCharacterId }
func (m Model) PartyId() uint32                { return m.partyId }
func (m Model) SkillId() skill.Id              { return m.skillId }
func (m Model) SkillLevel() byte               { return m.skillLevel }
func (m Model) Field() field.Model             { return m.fld }
func (m Model) TownMapId() _map.Id             { return m.townMapId }
func (m Model) Slot() byte                     { return m.slot }
func (m Model) TownPortalId() uint32           { return m.townPortalId }
func (m Model) AreaX() point.X                 { return m.areaX }
func (m Model) AreaY() point.Y                 { return m.areaY }
func (m Model) TownX() point.X                 { return m.townX }
func (m Model) TownY() point.Y                 { return m.townY }
func (m Model) DeployTime() time.Time          { return m.deployTime }
func (m Model) ExpiresAt() time.Time           { return m.expiresAt }

func (m Model) Reslot(slot byte, townPortalId uint32, townX point.X, townY point.Y) Model {
	return Clone(m).SetSlot(slot).SetTownPortalId(townPortalId).SetTownX(townX).SetTownY(townY).Build()
}
