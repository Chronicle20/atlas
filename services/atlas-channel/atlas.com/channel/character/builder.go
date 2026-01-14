package character

import (
	"atlas-channel/equipment"
	"atlas-channel/inventory"
	"atlas-channel/pet"
	"atlas-channel/character/skill"
	"errors"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
)

var (
	ErrInvalidId = errors.New("character id must be greater than 0")
)

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
	jobId              uint16
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
	mapId              _map.Id
	spawnPoint         uint32
	gm                 int
	x                  int16
	y                  int16
	stance             byte
	meso               uint32
	pets               []pet.Model
	equipment          equipment.Model
	inventory          inventory.Model
	skills             []skill.Model
}

// NewModelBuilder creates a new builder instance
func NewModelBuilder() *modelBuilder {
	return &modelBuilder{}
}

// CloneModel creates a builder initialized with the Model's values
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
		mapId:              m.mapId,
		spawnPoint:         m.spawnPoint,
		gm:                 m.gm,
		x:                  m.x,
		y:                  m.y,
		stance:             m.stance,
		meso:               m.meso,
		pets:               m.pets,
		equipment:          m.equipment,
		inventory:          m.inventory,
		skills:             m.skills,
	}
}

func (b *modelBuilder) SetId(v uint32) *modelBuilder           { b.id = v; return b }
func (b *modelBuilder) SetAccountId(v uint32) *modelBuilder    { b.accountId = v; return b }
func (b *modelBuilder) SetWorldId(v world.Id) *modelBuilder    { b.worldId = v; return b }
func (b *modelBuilder) SetName(v string) *modelBuilder         { b.name = v; return b }
func (b *modelBuilder) SetGender(v byte) *modelBuilder         { b.gender = v; return b }
func (b *modelBuilder) SetSkinColor(v byte) *modelBuilder      { b.skinColor = v; return b }
func (b *modelBuilder) SetFace(v uint32) *modelBuilder         { b.face = v; return b }
func (b *modelBuilder) SetHair(v uint32) *modelBuilder         { b.hair = v; return b }
func (b *modelBuilder) SetLevel(v byte) *modelBuilder          { b.level = v; return b }
func (b *modelBuilder) SetJobId(v uint16) *modelBuilder        { b.jobId = v; return b }
func (b *modelBuilder) SetStrength(v uint16) *modelBuilder     { b.strength = v; return b }
func (b *modelBuilder) SetDexterity(v uint16) *modelBuilder    { b.dexterity = v; return b }
func (b *modelBuilder) SetIntelligence(v uint16) *modelBuilder { b.intelligence = v; return b }
func (b *modelBuilder) SetLuck(v uint16) *modelBuilder         { b.luck = v; return b }
func (b *modelBuilder) SetHp(v uint16) *modelBuilder           { b.hp = v; return b }
func (b *modelBuilder) SetMaxHp(v uint16) *modelBuilder        { b.maxHp = v; return b }
func (b *modelBuilder) SetMp(v uint16) *modelBuilder           { b.mp = v; return b }
func (b *modelBuilder) SetMaxMp(v uint16) *modelBuilder        { b.maxMp = v; return b }
func (b *modelBuilder) SetHpMpUsed(v int) *modelBuilder        { b.hpMpUsed = v; return b }
func (b *modelBuilder) SetAp(v uint16) *modelBuilder           { b.ap = v; return b }
func (b *modelBuilder) SetSp(v string) *modelBuilder           { b.sp = v; return b }
func (b *modelBuilder) SetExperience(v uint32) *modelBuilder   { b.experience = v; return b }
func (b *modelBuilder) SetFame(v int16) *modelBuilder          { b.fame = v; return b }
func (b *modelBuilder) SetGachaponExperience(v uint32) *modelBuilder {
	b.gachaponExperience = v
	return b
}
func (b *modelBuilder) SetMapId(v _map.Id) *modelBuilder              { b.mapId = v; return b }
func (b *modelBuilder) SetSpawnPoint(v uint32) *modelBuilder          { b.spawnPoint = v; return b }
func (b *modelBuilder) SetGm(v int) *modelBuilder                     { b.gm = v; return b }
func (b *modelBuilder) SetMeso(v uint32) *modelBuilder                { b.meso = v; return b }
func (b *modelBuilder) SetPets(v []pet.Model) *modelBuilder           { b.pets = v; return b }
func (b *modelBuilder) SetEquipment(v equipment.Model) *modelBuilder  { b.equipment = v; return b }
func (b *modelBuilder) SetInventory(v inventory.Model) *modelBuilder  { b.inventory = v; return b }
func (b *modelBuilder) SetSkills(v []skill.Model) *modelBuilder       { b.skills = v; return b }

// Build creates a new Model instance with validation
func (b *modelBuilder) Build() (Model, error) {
	if b.id == 0 {
		return Model{}, ErrInvalidId
	}
	return Model{
		id:                 b.id,
		accountId:          b.accountId,
		worldId:            b.worldId,
		name:               b.name,
		gender:             b.gender,
		skinColor:          b.skinColor,
		face:               b.face,
		hair:               b.hair,
		level:              b.level,
		jobId:              b.jobId,
		strength:           b.strength,
		dexterity:          b.dexterity,
		intelligence:       b.intelligence,
		luck:               b.luck,
		hp:                 b.hp,
		maxHp:              b.maxHp,
		mp:                 b.mp,
		maxMp:              b.maxMp,
		hpMpUsed:           b.hpMpUsed,
		ap:                 b.ap,
		sp:                 b.sp,
		experience:         b.experience,
		fame:               b.fame,
		gachaponExperience: b.gachaponExperience,
		mapId:              b.mapId,
		spawnPoint:         b.spawnPoint,
		gm:                 b.gm,
		x:                  b.x,
		y:                  b.y,
		stance:             b.stance,
		meso:               b.meso,
		pets:               b.pets,
		equipment:          b.equipment,
		inventory:          b.inventory,
		skills:             b.skills,
	}, nil
}

// MustBuild creates a new Model instance, panicking on validation error
func (b *modelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
