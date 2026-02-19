package history

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	// StartSession creates a new session record when a character logs in
	StartSession(characterId uint32, ch channel.Model) (Model, error)

	// EndSession closes the active session for a character
	EndSession(characterId uint32) error

	// GetActiveSession returns the current active session for a character, if any
	GetActiveSession(characterId uint32) (Model, error)

	// GetSessionsSince returns all sessions since the given timestamp
	GetSessionsSince(characterId uint32, since time.Time) ([]Model, error)

	// GetSessionsInRange returns all sessions that overlap with the given time range
	GetSessionsInRange(characterId uint32, start, end time.Time) ([]Model, error)

	// ComputePlaytimeSince computes total playtime since the given timestamp
	ComputePlaytimeSince(characterId uint32, since time.Time) (time.Duration, error)

	// ComputePlaytimeInRange computes total playtime within the given time range
	ComputePlaytimeInRange(characterId uint32, start, end time.Time) (time.Duration, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
}

func (p *ProcessorImpl) StartSession(characterId uint32, ch channel.Model) (Model, error) {
	// First, close any existing active session (safety check)
	_ = closeSession(p.db.WithContext(p.ctx), characterId)

	// Create new session
	m, err := createSession(p.db.WithContext(p.ctx), p.t.Id(), characterId, ch)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to create session for character [%d].", characterId)
		return Model{}, err
	}

	p.l.Debugf("Started session [%d] for character [%d] on world [%d] channel [%d].", m.Id(), characterId, ch.WorldId(), ch.Id())
	return m, nil
}

func (p *ProcessorImpl) EndSession(characterId uint32) error {
	err := closeSession(p.db.WithContext(p.ctx), characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to end session for character [%d].", characterId)
		return err
	}

	p.l.Debugf("Ended session for character [%d].", characterId)
	return nil
}

func (p *ProcessorImpl) GetActiveSession(characterId uint32) (Model, error) {
	return getActiveSession(p.db.WithContext(p.ctx), characterId)
}

func (p *ProcessorImpl) GetSessionsSince(characterId uint32, since time.Time) ([]Model, error) {
	return getSessionsSince(p.db.WithContext(p.ctx), characterId, since)
}

func (p *ProcessorImpl) GetSessionsInRange(characterId uint32, start, end time.Time) ([]Model, error) {
	return getSessionsInRange(p.db.WithContext(p.ctx), characterId, start, end)
}

func (p *ProcessorImpl) ComputePlaytimeSince(characterId uint32, since time.Time) (time.Duration, error) {
	sessions, err := p.GetSessionsSince(characterId, since)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	var total time.Duration
	for _, session := range sessions {
		total += session.OverlapsWith(since, now)
	}

	return total, nil
}

func (p *ProcessorImpl) ComputePlaytimeInRange(characterId uint32, start, end time.Time) (time.Duration, error) {
	sessions, err := p.GetSessionsInRange(characterId, start, end)
	if err != nil {
		return 0, err
	}

	var total time.Duration
	for _, session := range sessions {
		total += session.OverlapsWith(start, end)
	}

	return total, nil
}
