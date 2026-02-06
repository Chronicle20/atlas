package skill

import (
	"time"

	"github.com/Chronicle20/atlas-constants/job"
	"github.com/Chronicle20/atlas-constants/skill"
)

type Model struct {
	id                skill.Id
	level             byte
	masterLevel       byte
	expiration        time.Time
	cooldownExpiresAt time.Time
}

func (m Model) Id() skill.Id {
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
	if j, ok := job.FromSkillId(m.id); ok {
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
