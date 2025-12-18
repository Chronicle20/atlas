package guild

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor defines the interface for guild operations
type Processor interface {
	// GetByMemberId retrieves a guild by member ID
	GetByMemberId(decorators ...model.Decorator[Model]) func(memberId uint32) (Model, error)

	// IsLeader checks if a character is a guild leader
	IsLeader(characterId uint32) (bool, error)

	// HasGuild checks if a character has a guild
	HasGuild(characterId uint32) (bool, error)
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new guild processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

// GetByMemberId retrieves a guild by member ID
func (p *ProcessorImpl) GetByMemberId(decorators ...model.Decorator[Model]) func(memberId uint32) (Model, error) {
	return func(memberId uint32) (Model, error) {
		mp := byMemberIdProvider(p.l, p.ctx, memberId)
		return model.Map(model.Decorate(decorators))(mp)()
	}
}

// IsLeader checks if a character is a guild leader
func (p *ProcessorImpl) IsLeader(characterId uint32) (bool, error) {
	g, err := p.GetByMemberId()(characterId)
	if err != nil {
		return false, err
	}
	return g.LeaderId() == characterId, nil
}

// HasGuild checks if a character has a guild
func (p *ProcessorImpl) HasGuild(characterId uint32) (bool, error) {
	g, err := p.GetByMemberId()(characterId)
	if err != nil {
		return false, err
	}
	return g.Id() != 0, nil
}

// byMemberIdProvider creates a provider for guilds by member ID
func byMemberIdProvider(l logrus.FieldLogger, ctx context.Context, memberId uint32) model.Provider[Model] {
	return func() (Model, error) {
		models, err := requests.SliceProvider[RestModel, Model](l, ctx)(requestByMemberId(memberId), Extract, model.Filters[Model]())()
		if err != nil {
			return Model{}, err
		}
		if len(models) == 0 {
			return Model{}, nil
		}
		return models[0], nil
	}
}
