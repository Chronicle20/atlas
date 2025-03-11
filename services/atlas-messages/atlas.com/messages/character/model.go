package character

import (
	"atlas-messages/character/skill"
	"strconv"
	"strings"
)

type Model struct {
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
	skills             []skill.Model
}

func (m Model) Gm() bool {
	return m.gm == 1
}

func (m Model) Rank() uint32 {
	return 0
}

func (m Model) RankMove() uint32 {
	return 0
}

func (m Model) JobRank() uint32 {
	return 0
}

func (m Model) JobRankMove() uint32 {
	return 0
}

func (m Model) Gender() byte {
	return m.gender
}

func (m Model) SkinColor() byte {
	return m.skinColor
}

func (m Model) Face() uint32 {
	return m.face
}

func (m Model) Hair() uint32 {
	return m.hair
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Name() string {
	return m.name
}

func (m Model) Level() byte {
	return m.level
}

func (m Model) JobId() uint16 {
	return m.jobId
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

func (m Model) Hp() uint16 {
	return m.hp
}

func (m Model) MaxHp() uint16 {
	return m.maxHp
}

func (m Model) Mp() uint16 {
	return m.mp
}

func (m Model) MaxMp() uint16 {
	return m.maxMp
}

func (m Model) Ap() uint16 {
	return m.ap
}

func (m Model) HasSPTable() bool {
	switch m.jobId {
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

func (m Model) Sp() []uint16 {
	s := strings.Split(m.sp, ",")
	var sps = make([]uint16, 0)
	for _, x := range s {
		sp, err := strconv.ParseUint(x, 10, 16)
		if err == nil {
			sps = append(sps, uint16(sp))
		}
	}
	return sps
}

func (m Model) RemainingSp() uint16 {
	return m.Sp()[m.skillBook()]
}

func (m Model) skillBook() uint16 {
	if m.jobId >= 2210 && m.jobId <= 2218 {
		return m.jobId - 2209
	}
	return 0
}

func (m Model) Experience() uint32 {
	return m.experience
}

func (m Model) Fame() int16 {
	return m.fame
}

func (m Model) GachaponExperience() uint32 {
	return m.gachaponExperience
}

func (m Model) MapId() uint32 {
	return m.mapId
}

func (m Model) SpawnPoint() byte {
	return 0
}

func (m Model) AccountId() uint32 {
	return m.accountId
}

func (m Model) Meso() uint32 {
	return m.meso
}

func (m Model) X() int16 {
	return 0
}

func (m Model) Y() int16 {
	return 0
}

func (m Model) Stance() byte {
	return 0
}

func (m Model) WorldId() byte {
	return m.worldId
}

func (m Model) SetSkills(ms []skill.Model) Model {
	return Model{
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
		skills:             ms,
	}
}

func (m Model) Skills() []skill.Model {
	return m.skills
}
