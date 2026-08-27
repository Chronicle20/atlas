package character

import (
	"atlas-pets/inventory"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

func Clone(m Model) *Builder {
	return &Builder{
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
		stance:             m.stance,
		meso:               m.meso,
		inventory:          m.inventory,
	}
}

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
	x                  int16
	y                  int16
	stance             byte
	meso               uint32
	inventory          inventory.Model
}

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
func (b *Builder) SetX(x int16) *Builder                   { b.x = x; return b }
func (b *Builder) SetY(y int16) *Builder                   { b.y = y; return b }
func (b *Builder) SetMeso(v uint32) *Builder               { b.meso = v; return b }
func (b *Builder) SetInventory(v inventory.Model) *Builder { b.inventory = v; return b }

func (b *Builder) Build() (Model, error) {
	if b.id == 0 {
		return Model{}, errors.New("id is required")
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
		stance:             b.stance,
		meso:               b.meso,
		inventory:          b.inventory,
	}, nil
}
