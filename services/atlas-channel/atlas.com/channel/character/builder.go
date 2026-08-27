package character

import (
	"atlas-channel/character/skill"
	"atlas-channel/equipment"
	"atlas-channel/inventory"
	"atlas-channel/monsterbook"
	"atlas-channel/party"
	"atlas-channel/pet"
	"atlas-channel/quest"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

var ErrInvalidId = errors.New("character id must be greater than 0")

type builder struct {
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
	x                  int16
	y                  int16
	fh                 int16
	stance             byte
	meso               uint32
	pets               []pet.Model
	equipment          equipment.Model
	inventory          inventory.Model
	skills             []skill.Model
	quests             []quest.Model
	party              party.Model
	monsterBook        monsterbook.Model
}

// NewBuilder creates a new builder instance
func NewBuilder() *builder {
	return &builder{}
}

// CloneModel creates a builder initialized with the Model's values
func CloneModel(m Model) *builder {
	return &builder{
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
		x:                  m.x,
		y:                  m.y,
		fh:                 m.fh,
		stance:             m.stance,
		meso:               m.meso,
		pets:               m.pets,
		equipment:          m.equipment,
		inventory:          m.inventory,
		skills:             m.skills,
		quests:             m.quests,
		party:              m.party,
		monsterBook:        m.monsterBook,
	}
}

func (b *builder) SetId(v uint32) *builder           { b.id = v; return b }
func (b *builder) SetAccountId(v uint32) *builder    { b.accountId = v; return b }
func (b *builder) SetWorldId(v world.Id) *builder    { b.worldId = v; return b }
func (b *builder) SetName(v string) *builder         { b.name = v; return b }
func (b *builder) SetGender(v byte) *builder         { b.gender = v; return b }
func (b *builder) SetSkinColor(v byte) *builder      { b.skinColor = v; return b }
func (b *builder) SetFace(v uint32) *builder         { b.face = v; return b }
func (b *builder) SetHair(v uint32) *builder         { b.hair = v; return b }
func (b *builder) SetLevel(v byte) *builder          { b.level = v; return b }
func (b *builder) SetJobId(v job.Id) *builder        { b.jobId = v; return b }
func (b *builder) SetStrength(v uint16) *builder     { b.strength = v; return b }
func (b *builder) SetDexterity(v uint16) *builder    { b.dexterity = v; return b }
func (b *builder) SetIntelligence(v uint16) *builder { b.intelligence = v; return b }
func (b *builder) SetLuck(v uint16) *builder         { b.luck = v; return b }
func (b *builder) SetHp(v uint16) *builder           { b.hp = v; return b }
func (b *builder) SetMaxHp(v uint16) *builder        { b.maxHp = v; return b }
func (b *builder) SetMp(v uint16) *builder           { b.mp = v; return b }
func (b *builder) SetMaxMp(v uint16) *builder        { b.maxMp = v; return b }
func (b *builder) SetHpMpUsed(v int) *builder        { b.hpMpUsed = v; return b }
func (b *builder) SetAp(v uint16) *builder           { b.ap = v; return b }
func (b *builder) SetSp(v string) *builder           { b.sp = v; return b }
func (b *builder) SetExperience(v uint32) *builder   { b.experience = v; return b }
func (b *builder) SetFame(v int16) *builder          { b.fame = v; return b }
func (b *builder) SetGachaponExperience(v uint32) *builder {
	b.gachaponExperience = v
	return b
}
func (b *builder) SetSpawnPoint(v uint32) *builder         { b.spawnPoint = v; return b }
func (b *builder) SetGm(v int) *builder                    { b.gm = v; return b }
func (b *builder) SetX(v int16) *builder                   { b.x = v; return b }
func (b *builder) SetY(v int16) *builder                   { b.y = v; return b }
func (b *builder) SetMeso(v uint32) *builder               { b.meso = v; return b }
func (b *builder) SetPets(v []pet.Model) *builder          { b.pets = v; return b }
func (b *builder) SetEquipment(v equipment.Model) *builder { b.equipment = v; return b }
func (b *builder) SetInventory(v inventory.Model) *builder { b.inventory = v; return b }
func (b *builder) SetSkills(v []skill.Model) *builder      { b.skills = v; return b }
func (b *builder) SetQuests(v []quest.Model) *builder      { b.quests = v; return b }
func (b *builder) SetParty(v party.Model) *builder         { b.party = v; return b }
func (b *builder) SetMonsterBook(v monsterbook.Model) *builder {
	b.monsterBook = v
	return b
}

// Build creates a new Model instance with validation
func (b *builder) Build() (Model, error) {
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
		spawnPoint:         b.spawnPoint,
		gm:                 b.gm,
		x:                  b.x,
		y:                  b.y,
		fh:                 b.fh,
		stance:             b.stance,
		meso:               b.meso,
		pets:               b.pets,
		equipment:          b.equipment,
		inventory:          b.inventory,
		skills:             b.skills,
		quests:             b.quests,
		party:              b.party,
		monsterBook:        b.monsterBook,
	}, nil
}

// MustBuild creates a new Model instance, panicking on validation error
func (b *builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
