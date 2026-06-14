package skill

import (
	"atlas-doors/data/skill/effect"
	"context"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor is the interface for fetching skill data from atlas-data.
type Processor interface {
	GetById(skillId uint32) (Model, error)
	GetEffect(skillId uint32, level byte) (effect.Model, error)
}

// ProcessorImpl implements Processor using the atlas-data REST API.
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *ProcessorImpl {
	return &ProcessorImpl{l: l, ctx: ctx}
}

// GetById fetches the skill from atlas-data by skill id.
func (p *ProcessorImpl) GetById(skillId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(skillId), Extract)()
}

// GetEffect returns the effect for the given 1-based skill level.
// Level 0 returns an empty Model (no-op). Returns an error if the
// level exceeds the number of effects stored in atlas-data.
func (p *ProcessorImpl) GetEffect(skillId uint32, level byte) (effect.Model, error) {
	s, err := p.GetById(skillId)
	if err != nil {
		return effect.Model{}, err
	}
	if level == 0 {
		return effect.Model{}, nil
	}
	if len(s.Effects()) < int(level) {
		return effect.Model{}, errors.New("level out of bounds")
	}
	return s.Effects()[level-1], nil
}
