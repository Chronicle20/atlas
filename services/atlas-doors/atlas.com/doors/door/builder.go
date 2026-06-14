package door

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

type ModelBuilder struct {
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

func NewBuilder() *ModelBuilder { return &ModelBuilder{} }

func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
		areaDoorId: m.areaDoorId, townDoorId: m.townDoorId, ownerCharacterId: m.ownerCharacterId,
		partyId: m.partyId, skillId: m.skillId, skillLevel: m.skillLevel, fld: m.fld,
		townMapId: m.townMapId, slot: m.slot, townPortalId: m.townPortalId,
		areaX: m.areaX, areaY: m.areaY, townX: m.townX, townY: m.townY,
		deployTime: m.deployTime, expiresAt: m.expiresAt,
	}
}

func (b *ModelBuilder) SetAreaDoorId(v uint32) *ModelBuilder       { b.areaDoorId = v; return b }
func (b *ModelBuilder) SetTownDoorId(v uint32) *ModelBuilder       { b.townDoorId = v; return b }
func (b *ModelBuilder) SetOwnerCharacterId(v uint32) *ModelBuilder { b.ownerCharacterId = v; return b }
func (b *ModelBuilder) SetPartyId(v uint32) *ModelBuilder          { b.partyId = v; return b }
func (b *ModelBuilder) SetSkillId(v uint32) *ModelBuilder          { b.skillId = v; return b }
func (b *ModelBuilder) SetSkillLevel(v byte) *ModelBuilder         { b.skillLevel = v; return b }
func (b *ModelBuilder) SetField(v field.Model) *ModelBuilder       { b.fld = v; return b }
func (b *ModelBuilder) SetTownMapId(v _map.Id) *ModelBuilder       { b.townMapId = v; return b }
func (b *ModelBuilder) SetSlot(v byte) *ModelBuilder               { b.slot = v; return b }
func (b *ModelBuilder) SetTownPortalId(v uint32) *ModelBuilder     { b.townPortalId = v; return b }
func (b *ModelBuilder) SetAreaX(v int16) *ModelBuilder             { b.areaX = v; return b }
func (b *ModelBuilder) SetAreaY(v int16) *ModelBuilder             { b.areaY = v; return b }
func (b *ModelBuilder) SetTownX(v int16) *ModelBuilder             { b.townX = v; return b }
func (b *ModelBuilder) SetTownY(v int16) *ModelBuilder             { b.townY = v; return b }
func (b *ModelBuilder) SetDeployTime(v time.Time) *ModelBuilder    { b.deployTime = v; return b }
func (b *ModelBuilder) SetExpiresAt(v time.Time) *ModelBuilder     { b.expiresAt = v; return b }

func (b *ModelBuilder) Build() Model {
	return Model{
		areaDoorId: b.areaDoorId, townDoorId: b.townDoorId, ownerCharacterId: b.ownerCharacterId,
		partyId: b.partyId, skillId: b.skillId, skillLevel: b.skillLevel, fld: b.fld,
		townMapId: b.townMapId, slot: b.slot, townPortalId: b.townPortalId,
		areaX: b.areaX, areaY: b.areaY, townX: b.townX, townY: b.townY,
		deployTime: b.deployTime, expiresAt: b.expiresAt,
	}
}
