package character

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor is the character REST client used by the mini-game validation
// ladder. Hp backs the alive check; Name is exposed for future use.
type Processor interface {
	GetById(characterId uint32) (Model, error)
	ByIdProvider(characterId uint32) model.Provider[Model]
	Hp(characterId uint32) (uint16, error)
	Name(characterId uint32) (string, error)
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

func (p *ProcessorImpl) ByIdProvider(characterId uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId), Extract)
}

func (p *ProcessorImpl) GetById(characterId uint32) (Model, error) {
	return p.ByIdProvider(characterId)()
}

func (p *ProcessorImpl) Hp(characterId uint32) (uint16, error) {
	m, err := p.GetById(characterId)
	if err != nil {
		return 0, err
	}
	return m.Hp(), nil
}

func (p *ProcessorImpl) Name(characterId uint32) (string, error) {
	m, err := p.GetById(characterId)
	if err != nil {
		return "", err
	}
	return m.Name(), nil
}
