package job

import (
	"github.com/Chronicle20/atlas-constants/skill"
	"math"
)

type Job struct {
	id        Id
	skills    []skill.Skill
	fourthJob bool
}

func (j Job) Id() Id {
	return j.id
}

func (j Job) Skills() []skill.Skill {
	return j.skills
}

func (j Job) Buffs() []skill.Skill {
	return j.skills
}

func (j Job) IsFourthJob() bool {
	return j.fourthJob
}

func IsA(jobId Id, referenceJobs ...Id) bool {
	is := false
	for _, referenceJobId := range referenceJobs {
		if Is(jobId, referenceJobId) {
			is = true
		}
	}
	return is
}

func Is(jobId Id, referenceJobId Id) bool {
	characterBranch := jobId / 10
	referenceBranch := referenceJobId / 10
	return characterBranch == referenceBranch && jobId >= referenceJobId || referenceBranch%10 == 0 && jobId/100 == referenceJobId/100
}

func FromSkillId(skillId skill.Id) (Job, bool) {
	jobId := IdFromSkillId(skillId)
	j, ok := Jobs[jobId]
	return j, ok
}

func IdFromSkillId(skillId skill.Id) Id {
	return Id(math.Floor(float64(skillId) / float64(10000)))
}

func IsFourthJob(jobId Id) bool {
	j, ok := Jobs[jobId]
	if !ok {
		return false
	}
	return j.IsFourthJob()
}

func IsBeginner(jobId Id) bool {
	return IsA(jobId, BeginnerId, NoblesseId, LegendId, EvanId)
}

func GetType(jobId Id) Type {
	return Type(jobId / 1000)
}

func IsCygnus(jobId Id) bool {
	return GetType(jobId) == TypeCygnus
}

func GetSkillBook(jobId Id) int {
	if jobId >= EvanStage2Id && jobId <= EvanStage10Id {
		return int(jobId - 2209)
	}
	return 0
}

func FromIndex(jobIndex uint32, subJobIndex uint32) Id {
	jobId := BeginnerId
	if jobIndex == 0 {
		jobId = NoblesseId
	} else if jobIndex == 1 {
		if subJobIndex == 0 {
			jobId = BeginnerId
		} else if subJobIndex == 1 {
			//jobId = job.BladeRecruit TODO
		}
	} else if jobIndex == 2 {
		jobId = LegendId
	} else if jobIndex == 3 {
		jobId = EvanId
	}
	return jobId
}
