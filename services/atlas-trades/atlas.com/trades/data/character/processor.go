package character

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor is the character REST client used by the trade validation ladder.
// The single-field accessors are conveniences over GetById; a caller that needs
// more than one of them should read the Model once rather than issuing a
// request per field.
type Processor interface {
	GetById(characterId character.Id) (Model, error)
	ByIdProvider(characterId character.Id) model.Provider[Model]
	Hp(characterId character.Id) (uint16, error)
	Level(characterId character.Id) (byte, error)
	Name(characterId character.Id) (string, error)
	Meso(characterId character.Id) (uint32, error)
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

func (p *ProcessorImpl) ByIdProvider(characterId character.Id) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId), Extract)
}

func (p *ProcessorImpl) GetById(characterId character.Id) (Model, error) {
	return p.ByIdProvider(characterId)()
}

func (p *ProcessorImpl) Hp(characterId character.Id) (uint16, error) {
	m, err := p.GetById(characterId)
	if err != nil {
		return 0, err
	}
	return m.Hp(), nil
}

func (p *ProcessorImpl) Level(characterId character.Id) (byte, error) {
	m, err := p.GetById(characterId)
	if err != nil {
		return 0, err
	}
	return m.Level(), nil
}

func (p *ProcessorImpl) Name(characterId character.Id) (string, error) {
	m, err := p.GetById(characterId)
	if err != nil {
		return "", err
	}
	return m.Name(), nil
}

func (p *ProcessorImpl) Meso(characterId character.Id) (uint32, error) {
	m, err := p.GetById(characterId)
	if err != nil {
		return 0, err
	}
	return m.Meso(), nil
}
