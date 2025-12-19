package character

import (
	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
)

type BuilderConfiguration struct {
	useStarting4AP           bool
	useAutoAssignStartersAP  bool
	defaultInventoryCapacity uint32
}

func NewBuilderConfiguration(useStarting4AP bool, useAutoAssignStartersAP bool, defaultInventoryCapacity uint32) BuilderConfiguration {
	return BuilderConfiguration{
		useStarting4AP:           useStarting4AP,
		useAutoAssignStartersAP:  useAutoAssignStartersAP,
		defaultInventoryCapacity: defaultInventoryCapacity,
	}
}

func (b *BuilderConfiguration) UseStarting4AP() bool {
	return b.useStarting4AP
}

func (b *BuilderConfiguration) UseAutoAssignStartersAP() bool {
	return b.useAutoAssignStartersAP
}

type Builder struct {
	accountId    uint32
	worldId      world.Id
	name         string
	level        byte
	strength     uint16
	dexterity    uint16
	intelligence uint16
	luck         uint16
	maxHp        uint16
	maxMp        uint16
	jobId        job.Id
	skinColor    byte
	gender       byte
	hair         uint32
	face         uint32
	ap           uint16
	mapId        _map.Id
	gm           int
}

func (b *Builder) SetJobId(jobId job.Id) *Builder {
	b.jobId = jobId
	return b
}

func (b *Builder) SetMapId(mapId _map.Id) *Builder {
	b.mapId = mapId
	return b
}

func (b *Builder) SetGm(gm int) *Builder {
	b.gm = gm
	return b
}

func (b *Builder) Build() Model {
	return Model{
		accountId:          b.accountId,
		worldId:            b.worldId,
		name:               b.name,
		level:              b.level,
		experience:         0,
		gachaponExperience: 0,
		strength:           b.strength,
		dexterity:          b.dexterity,
		intelligence:       b.intelligence,
		luck:               b.luck,
		hp:                 0,
		mp:                 0,
		maxHp:              b.maxHp,
		maxMp:              b.maxMp,
		meso:               0,
		hpMpUsed:           0,
		jobId:              b.jobId,
		skinColor:          b.skinColor,
		gender:             b.gender,
		fame:               0,
		hair:               b.hair,
		face:               b.face,
		ap:                 b.ap,
		sp:                 "",
		mapId:              b.mapId,
		spawnPoint:         0,
		gm:                 b.gm,
	}
}

func NewBuilder(c BuilderConfiguration, accountId uint32, worldId world.Id, name string, skinColor byte, gender byte, hair uint32, face uint32) *Builder {
	b := &Builder{
		accountId: accountId,
		worldId:   worldId,
		name:      name,
		level:     1,
		jobId:     0,
		skinColor: skinColor,
		gender:    gender,
		hair:      hair,
		face:      face,
	}

	if !c.UseStarting4AP() {
		if c.UseAutoAssignStartersAP() {
			b.strength = 12
			b.dexterity = 5
			b.intelligence = 4
			b.luck = 4
			b.ap = 0
		} else {
			b.strength = 4
			b.dexterity = 4
			b.intelligence = 4
			b.luck = 4
			b.ap = 9
		}
	} else {
		b.strength = 4
		b.dexterity = 4
		b.intelligence = 4
		b.luck = 4
		b.ap = 0
	}

	b.maxHp = 50
	b.maxMp = 5
	return b
}
