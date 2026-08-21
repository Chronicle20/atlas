package playernpc

import (
	"atlas-player-npcs/routing"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

// Builder builds a Model. Dir defaults to 1 (FR-4.6); SetX also derives
// rx0/rx1 (design §3.1) so a caller that only sets x gets the correct
// computed placement -- SetRX0/SetRX1 exist to let entity.Make restore the
// exact stored values on read without recomputing them.
type Builder struct {
	id             uuid.UUID
	characterId    uint32
	name           string
	worldId        byte
	mapId          uint32
	scriptId       uint32
	objectId       uint32
	gender         byte
	skin           byte
	face           uint32
	hair           uint32
	jobId          uint16
	x              int16
	cy             int16
	fh             uint16
	rx0            int16
	rx1            int16
	dir            byte
	step           byte
	worldRank      uint32
	overallRank    uint32
	worldJobRank   uint32
	overallJobRank uint32
	createdAt      time.Time
	updatedAt      time.Time
	equipment      []EquipmentModel
}

// NewBuilder creates a new Builder with dir defaulted to 1 (FR-4.6).
func NewBuilder() *Builder {
	return &Builder{
		dir: 1,
	}
}

func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetCharacterId(characterId uint32) *Builder {
	b.characterId = characterId
	return b
}

func (b *Builder) SetName(name string) *Builder {
	b.name = name
	return b
}

func (b *Builder) SetWorldId(worldId byte) *Builder {
	b.worldId = worldId
	return b
}

func (b *Builder) SetMapId(mapId uint32) *Builder {
	b.mapId = mapId
	return b
}

func (b *Builder) SetScriptId(scriptId uint32) *Builder {
	b.scriptId = scriptId
	return b
}

func (b *Builder) SetObjectId(objectId uint32) *Builder {
	b.objectId = objectId
	return b
}

func (b *Builder) SetGender(gender byte) *Builder {
	b.gender = gender
	return b
}

func (b *Builder) SetSkin(skin byte) *Builder {
	b.skin = skin
	return b
}

func (b *Builder) SetFace(face uint32) *Builder {
	b.face = face
	return b
}

func (b *Builder) SetHair(hair uint32) *Builder {
	b.hair = hair
	return b
}

// SetJobId stores the job CATEGORY (routing.JobCategory), per PRD §6 --
// (jobId / 100) * 100 -- not the raw wire job id.
func (b *Builder) SetJobId(jobId job.Id) *Builder {
	b.jobId = routing.JobCategory(jobId)
	return b
}

// setJobIdCategory sets the already-computed job category verbatim, for
// entity.Make to restore the stored value without recomputing it through
// routing.JobCategory (the entity already stores the category, not a raw
// job wire id).
func (b *Builder) setJobIdCategory(jobId uint16) *Builder {
	b.jobId = jobId
	return b
}

// SetX sets x and derives rx0 = x + 50, rx1 = x - 50 (design §3.1). Call
// SetRX0/SetRX1 afterward to override -- entity.Make does this to restore
// exactly what was stored rather than recompute it.
func (b *Builder) SetX(x int16) *Builder {
	b.x = x
	b.rx0 = x + 50
	b.rx1 = x - 50
	return b
}

func (b *Builder) SetCy(cy int16) *Builder {
	b.cy = cy
	return b
}

func (b *Builder) SetFh(fh uint16) *Builder {
	b.fh = fh
	return b
}

// SetRX0 overrides the rx0 derived by SetX. Used by entity.Make to restore
// the stored value verbatim.
func (b *Builder) SetRX0(rx0 int16) *Builder {
	b.rx0 = rx0
	return b
}

// SetRX1 overrides the rx1 derived by SetX. Used by entity.Make to restore
// the stored value verbatim.
func (b *Builder) SetRX1(rx1 int16) *Builder {
	b.rx1 = rx1
	return b
}

func (b *Builder) SetDir(dir byte) *Builder {
	b.dir = dir
	return b
}

// SetStep sets the grid/podium positioner step this NPC was placed at
// (design 5.1/5.2, Task 15's resolution of design 3.1's open question --
// see processor.go's package doc for the rationale).
func (b *Builder) SetStep(step byte) *Builder {
	b.step = step
	return b
}

func (b *Builder) SetWorldRank(worldRank uint32) *Builder {
	b.worldRank = worldRank
	return b
}

func (b *Builder) SetOverallRank(overallRank uint32) *Builder {
	b.overallRank = overallRank
	return b
}

func (b *Builder) SetWorldJobRank(worldJobRank uint32) *Builder {
	b.worldJobRank = worldJobRank
	return b
}

func (b *Builder) SetOverallJobRank(overallJobRank uint32) *Builder {
	b.overallJobRank = overallJobRank
	return b
}

func (b *Builder) SetCreatedAt(createdAt time.Time) *Builder {
	b.createdAt = createdAt
	return b
}

func (b *Builder) SetUpdatedAt(updatedAt time.Time) *Builder {
	b.updatedAt = updatedAt
	return b
}

// SetEquipment replaces the equipment child collection.
func (b *Builder) SetEquipment(equipment []EquipmentModel) *Builder {
	b.equipment = equipment
	return b
}

// AddEquipment appends a single equipment slot.
func (b *Builder) AddEquipment(equipment EquipmentModel) *Builder {
	b.equipment = append(b.equipment, equipment)
	return b
}

func (b *Builder) Build() (Model, error) {
	if err := b.validate(); err != nil {
		return Model{}, err
	}
	return Model{
		id:             b.id,
		characterId:    b.characterId,
		name:           b.name,
		worldId:        b.worldId,
		mapId:          b.mapId,
		scriptId:       b.scriptId,
		objectId:       b.objectId,
		gender:         b.gender,
		skin:           b.skin,
		face:           b.face,
		hair:           b.hair,
		jobId:          b.jobId,
		x:              b.x,
		cy:             b.cy,
		fh:             b.fh,
		rx0:            b.rx0,
		rx1:            b.rx1,
		dir:            b.dir,
		step:           b.step,
		worldRank:      b.worldRank,
		overallRank:    b.overallRank,
		worldJobRank:   b.worldJobRank,
		overallJobRank: b.overallJobRank,
		createdAt:      b.createdAt,
		updatedAt:      b.updatedAt,
		equipment:      b.equipment,
	}, nil
}

func (b *Builder) validate() error {
	if b.characterId == 0 {
		return errors.New("characterId is required")
	}
	if b.name == "" {
		return errors.New("name is required")
	}
	return nil
}

// EquipmentBuilder builds an EquipmentModel.
type EquipmentBuilder struct {
	id     uuid.UUID
	slot   int16
	itemId uint32
}

// NewEquipmentBuilder creates a new EquipmentBuilder.
func NewEquipmentBuilder() *EquipmentBuilder {
	return &EquipmentBuilder{}
}

func (b *EquipmentBuilder) SetId(id uuid.UUID) *EquipmentBuilder {
	b.id = id
	return b
}

func (b *EquipmentBuilder) SetSlot(slot int16) *EquipmentBuilder {
	b.slot = slot
	return b
}

func (b *EquipmentBuilder) SetItemId(itemId uint32) *EquipmentBuilder {
	b.itemId = itemId
	return b
}

func (b *EquipmentBuilder) Build() (EquipmentModel, error) {
	return EquipmentModel{
		id:     b.id,
		slot:   b.slot,
		itemId: b.itemId,
	}, nil
}
