package character

import (
	"atlas-character/skill"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
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
	gm           int
}

func (b *Builder) SetJobId(jobId job.Id) *Builder {
	b.jobId = jobId
	return b
}

func (b *Builder) SetGm(gm int) *Builder {
	b.gm = gm
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.accountId == 0 {
		return Model{}, errors.New("accountId is required")
	}
	if b.name == "" {
		return Model{}, errors.New("name is required")
	}

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
		spawnPoint:         0,
		gm:                 b.gm,
	}, nil
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

type modelBuilder struct {
	id                 uint32
	accountId          uint32
	worldId            world.Id
	name               string
	gender             byte
	skinColor          byte
	face               uint32
	hair               uint32
	level              byte
	jobId              job.Id
	strength           uint16
	dexterity          uint16
	intelligence       uint16
	luck               uint16
	hp                 uint16
	maxHp              uint16
	mp                 uint16
	maxMp              uint16
	hpMpUsed           int
	ap                 uint16
	sp                 string
	experience         uint32
	fame               int16
	gachaponExperience uint32
	spawnPoint         uint32
	gm                 int
	meso               uint32
	skills             []skill.Model
}

// NewEmptyBuilder creates a zero-valued builder for reconstructing a Model.
// The creation-flow builder is NewBuilder in builder.go; the two are distinct.
func NewEmptyBuilder() *modelBuilder {
	return &modelBuilder{}
}

func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
		id:                 m.id,
		accountId:          m.accountId,
		worldId:            m.worldId,
		name:               m.name,
		gender:             m.gender,
		skinColor:          m.skinColor,
		face:               m.face,
		hair:               m.hair,
		level:              m.level,
		jobId:              m.jobId,
		strength:           m.strength,
		dexterity:          m.dexterity,
		intelligence:       m.intelligence,
		luck:               m.luck,
		hp:                 m.hp,
		maxHp:              m.maxHp,
		mp:                 m.mp,
		maxMp:              m.maxMp,
		hpMpUsed:           m.hpMpUsed,
		ap:                 m.ap,
		sp:                 m.sp,
		experience:         m.experience,
		fame:               m.fame,
		gachaponExperience: m.gachaponExperience,
		spawnPoint:         m.spawnPoint,
		gm:                 m.gm,
		meso:               m.meso,
		skills:             m.skills,
	}
}

func (c *modelBuilder) SetId(id uint32) *modelBuilder {
	c.id = id
	return c
}

func (c *modelBuilder) SetAccountId(accountId uint32) *modelBuilder {
	c.accountId = accountId
	return c
}

func (c *modelBuilder) SetWorldId(worldId world.Id) *modelBuilder {
	c.worldId = worldId
	return c
}

func (c *modelBuilder) SetName(name string) *modelBuilder {
	c.name = name
	return c
}

func (c *modelBuilder) SetGender(gender byte) *modelBuilder {
	c.gender = gender
	return c
}

func (c *modelBuilder) SetSkinColor(skinColor byte) *modelBuilder {
	c.skinColor = skinColor
	return c
}

func (c *modelBuilder) SetFace(face uint32) *modelBuilder {
	c.face = face
	return c
}

func (c *modelBuilder) SetHair(hair uint32) *modelBuilder {
	c.hair = hair
	return c
}

func (c *modelBuilder) SetLevel(level byte) *modelBuilder {
	c.level = level
	return c
}

func (c *modelBuilder) SetJobId(jobId job.Id) *modelBuilder {
	c.jobId = jobId
	return c
}

func (c *modelBuilder) SetStrength(strength uint16) *modelBuilder {
	c.strength = strength
	return c
}

func (c *modelBuilder) SetDexterity(dexterity uint16) *modelBuilder {
	c.dexterity = dexterity
	return c
}

func (c *modelBuilder) SetIntelligence(intelligence uint16) *modelBuilder {
	c.intelligence = intelligence
	return c
}

func (c *modelBuilder) SetLuck(luck uint16) *modelBuilder {
	c.luck = luck
	return c
}

func (c *modelBuilder) SetHp(hp uint16) *modelBuilder {
	c.hp = hp
	return c
}

func (c *modelBuilder) SetMaxHp(maxHp uint16) *modelBuilder {
	c.maxHp = maxHp
	return c
}

func (c *modelBuilder) SetMp(mp uint16) *modelBuilder {
	c.mp = mp
	return c
}

func (c *modelBuilder) SetMaxMp(maxMp uint16) *modelBuilder {
	c.maxMp = maxMp
	return c
}

func (c *modelBuilder) SetAp(ap uint16) *modelBuilder {
	c.ap = ap
	return c
}

func (c *modelBuilder) SetSp(sp string) *modelBuilder {
	c.sp = sp
	return c
}

func (c *modelBuilder) SetExperience(experience uint32) *modelBuilder {
	c.experience = experience
	return c
}

func (c *modelBuilder) SetFame(fame int16) *modelBuilder {
	c.fame = fame
	return c
}

func (c *modelBuilder) SetGachaponExperience(gachaponExperience uint32) *modelBuilder {
	c.gachaponExperience = gachaponExperience
	return c
}

func (c *modelBuilder) SetSpawnPoint(spawnPoint uint32) *modelBuilder {
	c.spawnPoint = spawnPoint
	return c
}

func (c *modelBuilder) SetGm(gm int) *modelBuilder {
	c.gm = gm
	return c
}

func (c *modelBuilder) SetMeso(meso uint32) *modelBuilder {
	c.meso = meso
	return c
}

// Build enforces the reconstruction invariant: a model tied to a real
// account (accountId != 0) must carry a name. modelBuilder hydrates PARTIAL
// models across ~95 call sites -- DB rows, REST Extract, kafka create
// commands, decorator rebuilds, and test fixtures that legitimately set
// only a handful of fields (character/hp_mp_gain_test.go builds a model
// with only jobId and skills). The creation-path invariant used by Builder
// (accountId != 0 AND name != "") would reject those legitimate partials, so
// this is the strongest invariant that survives every construction site:
// every site that sets a real accountId also sets a name; the ones that
// leave accountId at zero legitimately leave name blank too. See
// docs/tasks/task-272-character-spawn-point-plumbing/builder-validation.md.
func (c *modelBuilder) Build() (Model, error) {
	if c.accountId != 0 && c.name == "" {
		return Model{}, errors.New("name is required when accountId is set")
	}
	return Model{
		id:                 c.id,
		accountId:          c.accountId,
		worldId:            c.worldId,
		name:               c.name,
		gender:             c.gender,
		skinColor:          c.skinColor,
		face:               c.face,
		hair:               c.hair,
		level:              c.level,
		jobId:              c.jobId,
		strength:           c.strength,
		dexterity:          c.dexterity,
		intelligence:       c.intelligence,
		luck:               c.luck,
		hp:                 c.hp,
		maxHp:              c.maxHp,
		mp:                 c.mp,
		maxMp:              c.maxMp,
		ap:                 c.ap,
		sp:                 c.sp,
		experience:         c.experience,
		fame:               c.fame,
		gachaponExperience: c.gachaponExperience,
		spawnPoint:         c.spawnPoint,
		gm:                 c.gm,
		meso:               c.meso,
		hpMpUsed:           c.hpMpUsed,
		skills:             c.skills,
	}, nil
}

func (c *modelBuilder) SetHpMpUsed(used int) *modelBuilder {
	c.hpMpUsed = used
	return c
}

func (c *modelBuilder) SetSkills(s []skill.Model) *modelBuilder {
	c.skills = s
	return c
}
