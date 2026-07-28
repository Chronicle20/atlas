package mock

import (
	"atlas-data/document"
	"atlas-data/job"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	GetSkillsForJobFunc func(jobId uint32) (job.RestModel, bool)
	RegisterFunc        func(s *document.Storage[string, job.RestModel], r model.Provider[[]job.RestModel]) error
	RegisterJobFunc     func(path string) (int, error)
}

var _ job.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetSkillsForJob(jobId uint32) (job.RestModel, bool) {
	if m.GetSkillsForJobFunc != nil {
		return m.GetSkillsForJobFunc(jobId)
	}
	return job.RestModel{}, false
}

func (m *ProcessorMock) Register(s *document.Storage[string, job.RestModel], r model.Provider[[]job.RestModel]) error {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(s, r)
	}
	return nil
}

func (m *ProcessorMock) RegisterJob(path string) (int, error) {
	if m.RegisterJobFunc != nil {
		return m.RegisterJobFunc(path)
	}
	return 0, nil
}
