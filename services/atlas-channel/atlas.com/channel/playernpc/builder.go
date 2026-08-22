package playernpc

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Builder constructs a Model. Tests and callers use it instead of a
// test-only constructor to keep the immutable-model pattern consistent.
type Builder struct {
	id             uuid.UUID
	characterId    uint32
	name           string
	worldId        world.Id
	mapId          _map.Id
	scriptId       uint32
	objectId       uint32
	gender         byte
	skin           byte
	face           uint32
	hair           uint32
	jobId          job.Id
	x              int16
	cy             int16
	fh             uint16
	rx0            int16
	rx1            int16
	dir            byte
	worldRank      uint32
	overallRank    uint32
	worldJobRank   uint32
	overallJobRank uint32
	equipment      []EquipmentModel
	deployedAt     time.Time
}

func NewBuilder(id uuid.UUID, characterId uint32) *Builder {
	return &Builder{
		id:          id,
		characterId: characterId,
	}
}

func (b *Builder) SetName(name string) *Builder {
	b.name = name
	return b
}

func (b *Builder) SetWorldId(worldId world.Id) *Builder {
	b.worldId = worldId
	return b
}

func (b *Builder) SetMapId(mapId _map.Id) *Builder {
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

func (b *Builder) SetJobId(jobId job.Id) *Builder {
	b.jobId = jobId
	return b
}

func (b *Builder) SetPosition(x int16, cy int16, fh uint16, rx0 int16, rx1 int16, dir byte) *Builder {
	b.x = x
	b.cy = cy
	b.fh = fh
	b.rx0 = rx0
	b.rx1 = rx1
	b.dir = dir
	return b
}

func (b *Builder) SetRanks(worldRank uint32, overallRank uint32, worldJobRank uint32, overallJobRank uint32) *Builder {
	b.worldRank = worldRank
	b.overallRank = overallRank
	b.worldJobRank = worldJobRank
	b.overallJobRank = overallJobRank
	return b
}

func (b *Builder) SetEquipment(equipment []EquipmentModel) *Builder {
	b.equipment = equipment
	return b
}

func (b *Builder) SetDeployedAt(deployedAt time.Time) *Builder {
	b.deployedAt = deployedAt
	return b
}

func (b *Builder) Build() Model {
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
		worldRank:      b.worldRank,
		overallRank:    b.overallRank,
		worldJobRank:   b.worldJobRank,
		overallJobRank: b.overallJobRank,
		equipment:      b.equipment,
		deployedAt:     b.deployedAt,
	}
}
