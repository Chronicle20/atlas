package character

import (
	"atlas-character/equipment"
	"atlas-character/inventory"
	"atlas-character/job"
	"strconv"
	"strings"
)

type Model struct {
	id                 uint32
	accountId          uint32
	worldId            byte
	name               string
	level              byte
	experience         uint32
	gachaponExperience uint32
	strength           uint16
	dexterity          uint16
	intelligence       uint16
	luck               uint16
	hp                 uint16
	mp                 uint16
	maxHp              uint16
	maxMp              uint16
	meso               uint32
	hpMpUsed           int
	jobId              uint16
	skinColor          byte
	gender             byte
	fame               int16
	hair               uint32
	face               uint32
	ap                 uint16
	sp                 string
	mapId              uint32
	spawnPoint         uint32
	gm                 int
	equipment          equipment.Model
	inventory          inventory.Model
}

func (m Model) HP() uint16 {
	return m.hp
}

func (m Model) MaxHP() uint16 {
	return m.maxHp
}

func (m Model) MP() uint16 {
	return m.mp
}

func (m Model) MaxMP() uint16 {
	return m.maxMp
}

func (m Model) Strength() uint16 {
	return m.strength
}

func (m Model) Dexterity() uint16 {
	return m.dexterity
}

func (m Model) Intelligence() uint16 {
	return m.intelligence
}

func (m Model) Luck() uint16 {
	return m.luck
}

func (m Model) JobId() uint16 {
	return m.jobId
}

func (m Model) Level() byte {
	return m.level
}

func (m Model) MaxClassLevel() byte {
	if m.Cygnus() {
		return 120
	} else {
		return 200
	}
}

func (m Model) Cygnus() bool {
	return m.JobType() == 1
}

func (m Model) JobType() uint16 {
	return m.jobId / 1000
}

func (m Model) Experience() uint32 {
	return m.experience
}

func (m Model) MapId() uint32 {
	return m.mapId
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) IsBeginner() bool {
	return m.jobId == job.Beginner || m.jobId == job.Noblesse || m.jobId == job.Legend
}

func (m Model) AP() uint16 {
	return m.ap
}

func (m Model) SP(i int) uint32 {
	sps := m.SPs()
	if len(sps) == 0 || i >= len(sps) {
		return 0
	}
	return sps[i]
}

func (m Model) SPs() []uint32 {
	sps := strings.Split(m.sp, ",")
	r := make([]uint32, 0)
	for _, sp := range sps {
		i, err := strconv.Atoi(sp)
		if err != nil {
			return r
		}
		r = append(r, uint32(i))
	}
	return r
}

func (m Model) SpawnPoint() uint32 {
	return m.spawnPoint
}

func (m Model) AccountId() uint32 {
	return m.accountId
}

func (m Model) WorldId() byte {
	return m.worldId
}

func (m Model) Name() string {
	return m.name
}

func (m Model) GachaponExperience() uint32 {
	return m.gachaponExperience
}

func (m Model) Meso() uint32 {
	return m.meso
}

func (m Model) SkinColor() byte {
	return m.skinColor
}

func (m Model) Gender() byte {
	return m.gender
}

func (m Model) Fame() int16 {
	return m.fame
}

func (m Model) Hair() uint32 {
	return m.hair
}

func (m Model) Face() uint32 {
	return m.face
}

func (m Model) SPString() string {
	return m.sp
}

func (m Model) GM() int {
	return m.gm
}

func (m Model) HPMPUsed() int {
	return m.hpMpUsed
}

func (m Model) GetEquipment() equipment.Model {
	return m.equipment
}

type modelBuilder struct {
	id                 uint32
	accountId          uint32
	worldId            byte
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
	mapId              uint32
	spawnPoint         uint32
	gm                 int
	meso               uint32
	equipment          equipment.Model
	inventory          inventory.Model
}

func NewModelBuilder() *modelBuilder {
	return &modelBuilder{equipment: equipment.NewModel()}
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
		mapId:              m.mapId,
		spawnPoint:         m.spawnPoint,
		gm:                 m.gm,
		meso:               m.meso,
		equipment:          m.equipment,
		inventory:          m.inventory,
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

func (c *modelBuilder) SetWorldId(worldId byte) *modelBuilder {
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

func (c *modelBuilder) SetJobId(jobId uint16) *modelBuilder {
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

func (c *modelBuilder) SetMapId(mapId uint32) *modelBuilder {
	c.mapId = mapId
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

func (c *modelBuilder) Build() Model {
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
		mapId:              c.mapId,
		spawnPoint:         c.spawnPoint,
		gm:                 c.gm,
		meso:               c.meso,
		equipment:          c.equipment,
		inventory:          c.inventory,
	}
}

func (c *modelBuilder) SetHpMpUsed(used int) *modelBuilder {
	c.hpMpUsed = used
	return c
}

func (c *modelBuilder) SetInventory(i inventory.Model) *modelBuilder {
	c.inventory = i
	return c
}

func (c *modelBuilder) SetEquipment(e equipment.Model) *modelBuilder {
	c.equipment = e
	return c
}
