package validation

import (
	"context"

	"github.com/sirupsen/logrus"
)

type Processor interface {
	HasItem(characterId uint32, itemId uint32) (bool, error)
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

// HasItem checks if a character has at least one of the specified item
func (p *ProcessorImpl) HasItem(characterId uint32, itemId uint32) (bool, error) {
	conditions := []ConditionInput{
		{
			Type:        ItemCondition,
			Operator:    ">=",
			Value:       1,
			ReferenceId: itemId,
		},
	}

	result, err := requestValidation(characterId, conditions)(p.l, p.ctx)
	if err != nil {
		return false, err
	}

	return result.AllPassed(), nil
}
