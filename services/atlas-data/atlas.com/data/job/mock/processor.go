package mock

import (
	"atlas-data/job"
)

type ProcessorMock struct {
	GetSkillsForJobFunc func(jobId uint32) (job.RestModel, bool)
}

var _ job.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetSkillsForJob(jobId uint32) (job.RestModel, bool) {
	if m.GetSkillsForJobFunc != nil {
		return m.GetSkillsForJobFunc(jobId)
	}
	return job.RestModel{}, false
}
