package skill

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

type Model struct {
	id          uint32
	level       byte
	masterLevel byte
	expiration  time.Time
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

// IsFourthJob is left raw-Id-keyed (task-187 audit): Model is a
// value-receiver type with no tenant/version in scope to resolve through, so
// version-aware resolution isn't available here at all. It's also provably
// a no-op for the audit's one divergent job set: job.FromSkillId floors
// m.id/10000 to a wire job id and looks up job.Jobs[jobId].IsFourthJob() --
// for the divergent wire ids (500/510 Pirate/Brawler at v0.61+, GM/SuperGM
// at v0.48; 900/910 Gm/SuperGm), every candidate entry in job.Jobs has
// fourthJob=false, so which identity a divergent skill id "really" belongs
// to is unobservable through this predicate; leaving it Id-keyed changes no
// outcome across the provisioned versions.
func (m Model) IsFourthJob() bool {
	if j, ok := job.FromSkillId(skill.Id(m.id)); ok {
		return j.IsFourthJob()
	}
	return false
}
