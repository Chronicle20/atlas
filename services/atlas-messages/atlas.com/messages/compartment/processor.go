package compartment

import (
	"atlas-messages/kafka/message/compartment"
	"atlas-messages/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	RequestCreateItem(characterId uint32, templateId uint32, quantity uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *ProcessorImpl) RequestCreateItem(characterId uint32, templateId uint32, quantity uint32) error {
	inventoryType, ok := inventory.TypeFromItemId(item.Id(templateId))
	if !ok {
		return errors.New("invalid templateId")
	}
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestCreateAssetCommandProvider(characterId, inventoryType, templateId, quantity))
}
