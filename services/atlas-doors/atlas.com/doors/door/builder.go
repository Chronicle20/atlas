package door

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

type Builder struct {
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

func NewBuilder() *Builder { return &Builder{} }

func Clone(m Model) *Builder {
	return &Builder{
		areaDoorId: m.areaDoorId, townDoorId: m.townDoorId, ownerCharacterId: m.ownerCharacterId,
		partyId: m.partyId, skillId: m.skillId, skillLevel: m.skillLevel, fld: m.fld,
		townMapId: m.townMapId, slot: m.slot, townPortalId: m.townPortalId,
		areaX: m.areaX, areaY: m.areaY, townX: m.townX, townY: m.townY,
		deployTime: m.deployTime, expiresAt: m.expiresAt,
	}
}

func (b *Builder) SetAreaDoorId(v uint32) *Builder { b.areaDoorId = v; return b }
func (b *Builder) SetTownDoorId(v uint32) *Builder { b.townDoorId = v; return b }
func (b *Builder) SetOwnerCharacterId(v character.Id) *Builder {
	b.ownerCharacterId = v
	return b
}
func (b *Builder) SetPartyId(v uint32) *Builder       { b.partyId = v; return b }
func (b *Builder) SetSkillId(v skill.Id) *Builder     { b.skillId = v; return b }
func (b *Builder) SetSkillLevel(v byte) *Builder      { b.skillLevel = v; return b }
func (b *Builder) SetField(v field.Model) *Builder    { b.fld = v; return b }
func (b *Builder) SetTownMapId(v _map.Id) *Builder    { b.townMapId = v; return b }
func (b *Builder) SetSlot(v byte) *Builder            { b.slot = v; return b }
func (b *Builder) SetTownPortalId(v uint32) *Builder  { b.townPortalId = v; return b }
func (b *Builder) SetAreaX(v point.X) *Builder        { b.areaX = v; return b }
func (b *Builder) SetAreaY(v point.Y) *Builder        { b.areaY = v; return b }
func (b *Builder) SetTownX(v point.X) *Builder        { b.townX = v; return b }
func (b *Builder) SetTownY(v point.Y) *Builder        { b.townY = v; return b }
func (b *Builder) SetDeployTime(v time.Time) *Builder { b.deployTime = v; return b }
func (b *Builder) SetExpiresAt(v time.Time) *Builder  { b.expiresAt = v; return b }

func (b *Builder) Build() Model {
	return Model{
		areaDoorId: b.areaDoorId, townDoorId: b.townDoorId, ownerCharacterId: b.ownerCharacterId,
		partyId: b.partyId, skillId: b.skillId, skillLevel: b.skillLevel, fld: b.fld,
		townMapId: b.townMapId, slot: b.slot, townPortalId: b.townPortalId,
		areaX: b.areaX, areaY: b.areaY, townX: b.townX, townY: b.townY,
		deployTime: b.deployTime, expiresAt: b.expiresAt,
	}
}
