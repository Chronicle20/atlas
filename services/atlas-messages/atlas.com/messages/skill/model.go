package skill

import (
	"github.com/Chronicle20/atlas-constants/job"
	"github.com/Chronicle20/atlas-constants/skill"
	"time"
)

type Model struct {
	id                uint32
	level             byte
	masterLevel       byte
	expiration        time.Time
	cooldownExpiresAt time.Time
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Level() byte {
	return m.level
}

func (m Model) MasterLevel() byte {
	return m.masterLevel
}

func (m Model) Expiration() time.Time {
	return m.expiration
}

func (m Model) IsFourthJob() bool {
	if j, ok := job.FromSkillId(skill.Id(m.id)); ok {
		return j.IsFourthJob()
	}
	return false
}

func (m Model) OnCooldown() bool {
	return time.Now().Before(m.cooldownExpiresAt)
}

func (m Model) CooldownExpiresAt() time.Time {
	return m.cooldownExpiresAt
}
