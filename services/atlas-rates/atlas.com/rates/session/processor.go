package session

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetSessionsSince(characterId uint32, since time.Time) ([]SessionRestModel, error)
	ComputePlaytimeInRange(characterId uint32, start, end time.Time) (time.Duration, error)
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

// GetSessionsSince retrieves all sessions for a character since the given time
func (p *ProcessorImpl) GetSessionsSince(characterId uint32, since time.Time) ([]SessionRestModel, error) {
	return RequestSessionsSince(characterId, since.Unix())(p.l, p.ctx)
}

// ComputePlaytimeSince computes total playtime for a character since the given time

// ComputePlaytimeInRange computes total playtime within a specific time range
func (p *ProcessorImpl) ComputePlaytimeInRange(characterId uint32, start, end time.Time) (time.Duration, error) {
	sessions, err := p.GetSessionsSince(characterId, start)
	if err != nil {
		return 0, err
	}

	var total time.Duration
	for _, session := range sessions {
		total += session.OverlapsWith(start, end)
	}

	return total, nil
}
