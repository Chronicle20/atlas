package job

import (
	"context"

	constJob "github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetSkillsForJob(jobId uint32) (RestModel, bool)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetSkillsForJob(jobId uint32) (RestModel, bool) {
	j, ok := constJob.Jobs[constJob.Id(uint16(jobId))]
	if !ok {
		return RestModel{Id: jobId, Skills: []uint32{}}, false
	}
	skills := make([]uint32, 0, len(j.Skills()))
	for _, s := range j.Skills() {
		skills = append(skills, uint32(s.Id()))
	}
	return RestModel{Id: jobId, Skills: skills}, true
}
