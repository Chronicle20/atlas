package properties

import (
	"strconv"
	"strings"
)

type Model struct {
	id                 uint32
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
	ap                 uint16
	sp                 string
	experience         uint32
	fame               int16
	gachaponExperience uint32
	mapId              uint32
	spawnPoint         byte
	gm                 bool
	gmJob              bool
	rank               int
	rankMove           int
	jobRank            int
	jobRankMove        int
}

func (a Model) Gm() bool {
	return a.gm
}

func (a Model) GmJob() bool {
	return a.gmJob
}

func (a Model) Rank() int {
	return a.rank
}

func (a Model) RankMove() int {
	return a.rankMove
}

func (a Model) JobRank() int {
	return a.jobRank
}

func (a Model) JobRankMove() int {
	return a.jobRankMove
}

func (a Model) Id() uint32 {
	return a.id
}

func (a Model) Name() string {
	return a.name
}

func (a Model) Gender() byte {
	return a.gender
}

func (a Model) SkinColor() byte {
	return a.skinColor
}

func (a Model) Face() uint32 {
	return a.face
}

func (a Model) Hair() uint32 {
	return a.hair
}

func (a Model) Level() byte {
	return a.level
}

func (a Model) JobId() uint16 {
	return a.jobId
}

func (a Model) Strength() uint16 {
	return a.strength
}

func (a Model) Dexterity() uint16 {
	return a.dexterity
}

func (a Model) Intelligence() uint16 {
	return a.intelligence
}

func (a Model) Luck() uint16 {
	return a.luck
}

func (a Model) Hp() uint16 {
	return a.hp
}

func (a Model) MaxHp() uint16 {
	return a.maxHp
}

func (a Model) Mp() uint16 {
	return a.mp
}

func (a Model) MaxMp() uint16 {
	return a.maxMp
}

func (a Model) Ap() uint16 {
	return a.ap
}

func (a Model) HasSPTable() bool {
	switch a.jobId {
	case 2001:
		return true
	case 2200:
		return true
	case 2210:
		return true
	case 2211:
		return true
	case 2212:
		return true
	case 2213:
		return true
	case 2214:
		return true
	case 2215:
		return true
	case 2216:
		return true
	case 2217:
		return true
	case 2218:
		return true
	default:
		return false
	}
}

func (a Model) Sp() []uint16 {
	s := strings.Split(a.sp, ",")
	var sps = make([]uint16, 0)
	for _, x := range s {
		sp, err := strconv.ParseUint(x, 10, 16)
		if err == nil {
			sps = append(sps, uint16(sp))
		}
	}
	return sps
}

func (a Model) RemainingSp() uint16 {
	return a.Sp()[a.skillBook()]
}

func (a Model) skillBook() uint16 {
	if a.jobId >= 2210 && a.jobId <= 2218 {
		return a.jobId - 2209
	}
	return 0
}

func (a Model) Experience() uint32 {
	return a.experience
}

func (a Model) Fame() int16 {
	return a.fame
}

func (a Model) GachaponExperience() uint32 {
	return a.gachaponExperience
}

func (a Model) SpawnPoint() byte {
	return a.spawnPoint
}

func (a Model) WorldId() byte {
	return a.worldId
}

func (a Model) MapId() uint32 {
	return a.mapId
}

type builder struct {
	id                 uint32
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
	ap                 uint16
	sp                 string
	experience         uint32
	fame               int16
	gachaponExperience uint32
	mapId              uint32
	spawnPoint         byte
	gm                 bool
	gmJob              bool
	rank               int
	rankMove           int
	jobRank            int
	jobRankMove        int
}

func NewBuilder() *builder {
	return &builder{}
}

func (c *builder) SetId(id uint32) *builder {
	c.id = id
	return c
}

func (c *builder) SetWorldId(worldId byte) *builder {
	c.worldId = worldId
	return c
}

func (c *builder) SetName(name string) *builder {
	c.name = name
	return c
}

func (c *builder) SetGender(gender byte) *builder {
	c.gender = gender
	return c
}

func (c *builder) SetSkinColor(skinColor byte) *builder {
	c.skinColor = skinColor
	return c
}

func (c *builder) SetFace(face uint32) *builder {
	c.face = face
	return c
}

func (c *builder) SetHair(hair uint32) *builder {
	c.hair = hair
	return c
}

func (c *builder) SetLevel(level byte) *builder {
	c.level = level
	return c
}

func (c *builder) SetJobId(jobId uint16) *builder {
	c.jobId = jobId
	return c
}

func (c *builder) SetStrength(strength uint16) *builder {
	c.strength = strength
	return c
}

func (c *builder) SetDexterity(dexterity uint16) *builder {
	c.dexterity = dexterity
	return c
}

func (c *builder) SetIntelligence(intelligence uint16) *builder {
	c.intelligence = intelligence
	return c
}

func (c *builder) SetLuck(luck uint16) *builder {
	c.luck = luck
	return c
}

func (c *builder) SetHp(hp uint16) *builder {
	c.hp = hp
	return c
}

func (c *builder) SetMaxHp(maxHp uint16) *builder {
	c.maxHp = maxHp
	return c
}

func (c *builder) SetMp(mp uint16) *builder {
	c.mp = mp
	return c
}

func (c *builder) SetMaxMp(maxMp uint16) *builder {
	c.maxMp = maxMp
	return c
}

func (c *builder) SetAp(ap uint16) *builder {
	c.ap = ap
	return c
}

func (c *builder) SetSp(sp string) *builder {
	c.sp = sp
	return c
}

func (c *builder) SetExperience(experience uint32) *builder {
	c.experience = experience
	return c
}

func (c *builder) SetFame(fame int16) *builder {
	c.fame = fame
	return c
}

func (c *builder) SetGachaponExperience(gachaponExperience uint32) *builder {
	c.gachaponExperience = gachaponExperience
	return c
}

func (c *builder) SetMapId(mapId uint32) *builder {
	c.mapId = mapId
	return c
}

func (c *builder) SetSpawnPoint(spawnPoint byte) *builder {
	c.spawnPoint = spawnPoint
	return c
}

func (c *builder) SetGm(gm bool) *builder {
	c.gm = gm
	return c
}

func (c *builder) SetGmJob(gmJob bool) *builder {
	c.gmJob = gmJob
	return c
}

func (c *builder) SetRank(rank int) *builder {
	c.rank = rank
	return c
}

func (c *builder) SetRankMove(rankMove int) *builder {
	c.rankMove = rankMove
	return c
}

func (c *builder) SetJobRank(jobRank int) *builder {
	c.jobRank = jobRank
	return c
}

func (c *builder) SetJobRankMove(jobRankMove int) *builder {
	c.jobRankMove = jobRankMove
	return c
}

func (c *builder) Build() Model {
	return Model{
		id:                 c.id,
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
		gmJob:              c.gmJob,
		rank:               c.rank,
		rankMove:           c.rankMove,
		jobRank:            c.jobRank,
		jobRankMove:        c.jobRankMove,
	}
}
