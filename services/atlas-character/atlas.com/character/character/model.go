package character

import (
	"atlas-character/skill"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type Model struct {
	id                 uint32
	accountId          uint32
	worldId            world.Id
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
	jobId              job.Id
	skinColor          byte
	gender             byte
	fame               int16
	hair               uint32
	face               uint32
	ap                 uint16
	sp                 string
	spawnPoint         uint32
	gm                 int
	skills             []skill.Model
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

func (m Model) JobId() job.Id {
	return m.jobId
}

func (m Model) Level() byte {
	return m.level
}

// version-stable per task-187 audit (audit/README.md, divergences.csv):
// Cygnus (job type 1xxx) is not part of the audit's divergent job set (only
// wire 500/510/900/910 GM<->Pirate remap across the provisioned GMS
// versions) -- job.IsCygnus(m.jobId) is safe to leave raw-Id-keyed. Model is
// also a value receiver with no tenant in scope to resolve through.
func (m Model) MaxClassLevel() byte {
	if job.IsCygnus(m.jobId) {
		return 120
	} else {
		return 200
	}
}

func (m Model) Experience() uint32 {
	return m.experience
}

func (m Model) Id() uint32 {
	return m.id
}

// version-stable per task-187 audit (audit/README.md, divergences.csv):
// Beginner/Noblesse/Legend/Evan roots are not part of the audit's divergent
// job set -- job.IsBeginner(m.JobId()) is safe to leave raw-Id-keyed. Model
// is also a value receiver with no tenant in scope to resolve through.
func (m Model) IsBeginner() bool {
	return job.IsBeginner(m.JobId())
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

func (m Model) WorldId() world.Id {
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

func (m Model) HpMpUsed() int {
	return m.hpMpUsed
}

func (m Model) GetSkill(skillId uint32) skill.Model {
	for _, s := range m.skills {
		if s.Id() == skillId {
			return s
		}
	}
	return skill.Model{}
}

func (m Model) GetSkillLevel(skillId uint32) byte {
	for _, s := range m.skills {
		if s.Id() == skillId {
			return s.Level()
		}
	}
	return 0
}
