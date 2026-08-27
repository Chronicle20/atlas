package character

import (
	"atlas-login/equipment"
	"atlas-login/inventory"
	"atlas-login/pet"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Builder is used to construct a Model instance
type Builder struct {
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
	pets               []pet.Model
	equipment          equipment.Model
	inventory          inventory.Model
	rank               uint32
	rankMove           int32
	jobRank            uint32
	jobRankMove        int32
}

// NewBuilder creates a new Builder instance
func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) SetId(v uint32) *Builder           { b.id = v; return b }
func (b *Builder) SetAccountId(v uint32) *Builder    { b.accountId = v; return b }
func (b *Builder) SetWorldId(v world.Id) *Builder    { b.worldId = v; return b }
func (b *Builder) SetName(v string) *Builder         { b.name = v; return b }
func (b *Builder) SetGender(v byte) *Builder         { b.gender = v; return b }
func (b *Builder) SetSkinColor(v byte) *Builder      { b.skinColor = v; return b }
func (b *Builder) SetFace(v uint32) *Builder         { b.face = v; return b }
func (b *Builder) SetHair(v uint32) *Builder         { b.hair = v; return b }
func (b *Builder) SetLevel(v byte) *Builder          { b.level = v; return b }
func (b *Builder) SetJobId(v job.Id) *Builder        { b.jobId = v; return b }
func (b *Builder) SetStrength(v uint16) *Builder     { b.strength = v; return b }
func (b *Builder) SetDexterity(v uint16) *Builder    { b.dexterity = v; return b }
func (b *Builder) SetIntelligence(v uint16) *Builder { b.intelligence = v; return b }
func (b *Builder) SetLuck(v uint16) *Builder         { b.luck = v; return b }
func (b *Builder) SetHp(v uint16) *Builder           { b.hp = v; return b }
func (b *Builder) SetMaxHp(v uint16) *Builder        { b.maxHp = v; return b }
func (b *Builder) SetMp(v uint16) *Builder           { b.mp = v; return b }
func (b *Builder) SetMaxMp(v uint16) *Builder        { b.maxMp = v; return b }
func (b *Builder) SetHpMpUsed(v int) *Builder        { b.hpMpUsed = v; return b }
func (b *Builder) SetAp(v uint16) *Builder           { b.ap = v; return b }
func (b *Builder) SetSp(v string) *Builder           { b.sp = v; return b }
func (b *Builder) SetExperience(v uint32) *Builder   { b.experience = v; return b }
func (b *Builder) SetFame(v int16) *Builder          { b.fame = v; return b }
func (b *Builder) SetGachaponExperience(v uint32) *Builder {
	b.gachaponExperience = v
	return b
}
func (b *Builder) SetSpawnPoint(v uint32) *Builder         { b.spawnPoint = v; return b }
func (b *Builder) SetGm(v int) *Builder                    { b.gm = v; return b }
func (b *Builder) SetMeso(v uint32) *Builder               { b.meso = v; return b }
func (b *Builder) SetPets(v []pet.Model) *Builder          { b.pets = v; return b }
func (b *Builder) SetEquipment(v equipment.Model) *Builder { b.equipment = v; return b }
func (b *Builder) SetInventory(v inventory.Model) *Builder { b.inventory = v; return b }
func (b *Builder) SetRank(v uint32) *Builder               { b.rank = v; return b }
func (b *Builder) SetRankMove(v int32) *Builder            { b.rankMove = v; return b }
func (b *Builder) SetJobRank(v uint32) *Builder            { b.jobRank = v; return b }
func (b *Builder) SetJobRankMove(v int32) *Builder         { b.jobRankMove = v; return b }

func (b *Builder) Build() Model {
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
		meso:               b.meso,
		pets:               b.pets,
		equipment:          b.equipment,
		inventory:          b.inventory,
		rank:               b.rank,
		rankMove:           b.rankMove,
		jobRank:            b.jobRank,
		jobRankMove:        b.jobRankMove,
	}
}
