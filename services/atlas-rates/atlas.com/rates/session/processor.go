package session

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// sessionDrainPageSize is the page size used to drain the full since-filtered
// sessions collection from atlas-character. Playtime computation is
// semantically an "all rows" consumer: a single-page fetch would silently
// truncate to the oldest N sessions (the endpoint paginates ordered by
// login_time ASC) and undercount equipped playtime.
const sessionDrainPageSize = 250

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
	return requests.DrainProvider[SessionRestModel, SessionRestModel](p.l, p.ctx)(SessionsSinceUrl(characterId, since.Unix()), sessionDrainPageSize, Extract, model.Filters[SessionRestModel]())()
}

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
