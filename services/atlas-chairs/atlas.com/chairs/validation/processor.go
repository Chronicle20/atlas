package validation

import (
	"context"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{
		l:   l,
		ctx: ctx,
	}
}

// HasItem checks if a character has at least one of the specified item
func (p *Processor) HasItem(characterId uint32, itemId uint32) (bool, error) {
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
