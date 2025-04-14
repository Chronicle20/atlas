package compartment

import (
	"atlas-character-factory/kafka/message/compartment"
	"atlas-character-factory/kafka/producer"
	compartment2 "atlas-character-factory/kafka/producer/compartment"
	"context"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/sirupsen/logrus"
	"time"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) CreateAsset(characterId uint32, inventoryType inventory.Type, templateId uint32, quantity uint32, expiration time.Time, ownerId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(compartment2.CreateAssetCommandProvider(characterId, inventoryType, templateId, quantity, expiration, ownerId))
}

func (p *Processor) EquipAsset(characterId uint32, inventoryType inventory.Type, source int16, destination int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(compartment2.EquipAssetCommandProvider(characterId, inventoryType, source, destination))
}
