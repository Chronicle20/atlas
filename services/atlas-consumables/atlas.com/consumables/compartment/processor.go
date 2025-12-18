package compartment

import (
	"atlas-consumables/kafka/message/compartment"
	"atlas-consumables/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Reserves struct {
	Slot     int16
	ItemId   uint32
	Quantity int16
}

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

func (p *Processor) RequestReserve(transactionId uuid.UUID, characterId uint32, it inventory.Type, reserves []Reserves) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(requestReserveCommandProvider(transactionId, characterId, it, reserves))
}

func Consume(f func(l logrus.FieldLogger) func(ctx context.Context) error) message.Handler[compartment.StatusEvent[compartment.ReservedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e compartment.StatusEvent[compartment.ReservedEventBody]) {
		_ = f(l)(ctx)
	}
}

func (p *Processor) ConsumeItem(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(consumeCommandProvider(characterId, inventoryType, transactionId, slot))
}

func (p *Processor) DestroyItem(characterId uint32, inventoryType inventory.Type, slot int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(destroyCommandProvider(characterId, inventoryType, slot))
}

func (p *Processor) CancelItemReservation(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(cancelReservationCommandProvider(characterId, inventoryType, transactionId, slot))
}
