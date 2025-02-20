package character

import (
	"atlas-channel/character/equipment"
	"atlas-channel/character/inventory"
	"atlas-channel/character/skill"
	"atlas-channel/pet"
	"github.com/Chronicle20/atlas-constants/job"
	"github.com/Chronicle20/atlas-constants/world"
	"strconv"
	"strings"
)

type Model struct {
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
	mapId              uint32
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
	switch job.Id(m.jobId) {
	case job.EvanId:
		return true
	case job.EvanStage1Id:
		return true
	case job.EvanStage2Id:
		return true
	case job.EvanStage3Id:
		return true
	case job.EvanStage4Id:
		return true
	case job.EvanStage5Id:
		return true
	case job.EvanStage6Id:
		return true
	case job.EvanStage7Id:
		return true
	case job.EvanStage8Id:
		return true
	case job.EvanStage9Id:
		return true
	case job.EvanStage10Id:
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

func (m Model) Equipment() equipment.Model {
	return m.equipment
}

func (m Model) Pets() []pet.Model {
	return m.pets
}

func (m Model) AccountId() uint32 {
	return m.accountId
}

func (m Model) Meso() uint32 {
	return m.meso
}

func (m Model) Inventory() inventory.Model {
	return m.inventory
}

func (m Model) X() int16 {
	return m.x
}

func (m Model) Y() int16 {
	return m.y
}

func (m Model) Stance() byte {
	return m.stance
}

func (m Model) WorldId() world.Id {
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
		x:                  m.x,
		y:                  m.y,
		stance:             m.stance,
		meso:               m.meso,
		pets:               m.pets,
		equipment:          m.equipment,
		inventory:          m.inventory,
		skills:             ms,
	}
}

func (m Model) Skills() []skill.Model {
	return m.skills
}
