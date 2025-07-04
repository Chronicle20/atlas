package drop

import (
	"atlas-drops/drop"
	consumer2 "atlas-drops/kafka/consumer"
	messageDropKafka "atlas-drops/kafka/message/drop"
	"context"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("drop_command")(messageDropKafka.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(messageDropKafka.EnvCommandTopic)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleSpawn)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleSpawnFromCharacter)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestReservation)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleCancelReservation)))
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestPickUp)))
	}
}

func handleSpawn(l logrus.FieldLogger, ctx context.Context, c messageDropKafka.Command[messageDropKafka.CommandSpawnBody]) {
	if c.Type != messageDropKafka.CommandTypeSpawn {
		return
	}
	t := tenant.MustFromContext(ctx)
	mb := drop.NewModelBuilder(t, c.WorldId, c.ChannelId, c.MapId).
		SetItem(c.Body.ItemId, c.Body.Quantity).
		SetMeso(c.Body.Mesos).
		SetType(c.Body.DropType).
		SetPosition(c.Body.X, c.Body.Y).
		SetOwner(c.Body.OwnerId, c.Body.OwnerPartyId).
		SetDropper(c.Body.DropperId, c.Body.DropperX, c.Body.DropperY).
		SetPlayerDrop(c.Body.PlayerDrop)
	p := drop.NewProcessor(l, ctx)
	_, _ = p.SpawnAndEmit(mb)
}

func handleSpawnFromCharacter(l logrus.FieldLogger, ctx context.Context, c messageDropKafka.Command[messageDropKafka.CommandSpawnFromCharacterBody]) {
	if c.Type != messageDropKafka.CommandTypeSpawnFromCharacter {
		return
	}
	t := tenant.MustFromContext(ctx)
	mb := drop.NewModelBuilder(t, c.WorldId, c.ChannelId, c.MapId).
		SetItem(c.Body.ItemId, c.Body.Quantity).
		SetEquipmentId(c.Body.EquipmentId).
		SetMeso(c.Body.Mesos).
		SetType(c.Body.DropType).
		SetPosition(c.Body.X, c.Body.Y).
		SetOwner(c.Body.OwnerId, c.Body.OwnerPartyId).
		SetDropper(c.Body.DropperId, c.Body.DropperX, c.Body.DropperY).
		SetPlayerDrop(c.Body.PlayerDrop)
	p := drop.NewProcessor(l, ctx)
	_, _ = p.SpawnForCharacterAndEmit(mb)
}

func handleRequestReservation(l logrus.FieldLogger, ctx context.Context, c messageDropKafka.Command[messageDropKafka.CommandRequestReservationBody]) {
	if c.Type != messageDropKafka.CommandTypeRequestReservation {
		return
	}
	p := drop.NewProcessor(l, ctx)
	_, _ = p.ReserveAndEmit(c.TransactionId, c.WorldId, c.ChannelId, c.MapId, c.Body.DropId, c.Body.CharacterId, c.Body.PetSlot)
}

func handleCancelReservation(l logrus.FieldLogger, ctx context.Context, c messageDropKafka.Command[messageDropKafka.CommandCancelReservationBody]) {
	if c.Type != messageDropKafka.CommandTypeCancelReservation {
		return
	}
	p := drop.NewProcessor(l, ctx)
	_ = p.CancelReservationAndEmit(c.TransactionId, c.WorldId, c.ChannelId, c.MapId, c.Body.DropId, c.Body.CharacterId)
}

func handleRequestPickUp(l logrus.FieldLogger, ctx context.Context, c messageDropKafka.Command[messageDropKafka.CommandRequestPickUpBody]) {
	if c.Type != messageDropKafka.CommandTypeRequestPickUp {
		return
	}
	p := drop.NewProcessor(l, ctx)
	_, _ = p.GatherAndEmit(c.TransactionId, c.WorldId, c.ChannelId, c.MapId, c.Body.DropId, c.Body.CharacterId)
}
