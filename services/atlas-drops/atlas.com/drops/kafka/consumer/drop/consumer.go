package drop

import (
	"atlas-drops/data/foothold"
	"atlas-drops/drop"
	consumer2 "atlas-drops/kafka/consumer"
	messageDropKafka "atlas-drops/kafka/message/drop"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("drop_command")(messageDropKafka.EnvCommandTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(messageDropKafka.EnvCommandTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSpawn))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleSpawnFromCharacter))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestReservation))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleCancelReservation))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestPickUp))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleConsume))); err != nil {
			return err
		}
		return nil
	}
}

func handleSpawn(l logrus.FieldLogger, ctx context.Context, c messageDropKafka.Command[messageDropKafka.CommandSpawnBody]) {
	if c.Type != messageDropKafka.CommandTypeSpawn {
		return
	}
	t := tenant.MustFromContext(ctx)
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	mb := drop.NewModelBuilder(t, f).
		SetItem(c.Body.ItemId, c.Body.Quantity).
		SetMeso(c.Body.Mesos).
		SetType(c.Body.DropType).
		SetPosition(c.Body.X, c.Body.Y).
		SetOwner(c.Body.OwnerId, c.Body.OwnerPartyId).
		SetDropper(c.Body.DropperId, c.Body.DropperX, c.Body.DropperY).
		SetPlayerDrop(c.Body.PlayerDrop).
		SetStrength(c.Body.Strength).
		SetDexterity(c.Body.Dexterity).
		SetIntelligence(c.Body.Intelligence).
		SetLuck(c.Body.Luck).
		SetHp(c.Body.Hp).
		SetMp(c.Body.Mp).
		SetWeaponAttack(c.Body.WeaponAttack).
		SetMagicAttack(c.Body.MagicAttack).
		SetWeaponDefense(c.Body.WeaponDefense).
		SetMagicDefense(c.Body.MagicDefense).
		SetAccuracy(c.Body.Accuracy).
		SetAvoidability(c.Body.Avoidability).
		SetHands(c.Body.Hands).
		SetSpeed(c.Body.Speed).
		SetJump(c.Body.Jump).
		SetSlots(c.Body.Slots)
	p := drop.NewProcessor(l, ctx)
	_, _ = p.SpawnAndEmit(mb)
}

func handleSpawnFromCharacter(l logrus.FieldLogger, ctx context.Context, c messageDropKafka.Command[messageDropKafka.CommandSpawnFromCharacterBody]) {
	if c.Type != messageDropKafka.CommandTypeSpawnFromCharacter {
		return
	}
	t := tenant.MustFromContext(ctx)
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	// Snap the drop's resting position to the foothold directly below its
	// origin so it falls to the floor instead of hanging mid-air (e.g. when
	// dropped while on a rope/ladder). The dropper coordinates stay the
	// character's position, so the client animates the fall from origin to
	// floor. If nothing is below (map edge / bottomless pit), keep the
	// original y.
	restX, restY := c.Body.X, c.Body.Y
	if ly, ok := foothold.NewProcessor(l, ctx).LandingBelow(c.MapId, c.Body.X, c.Body.Y); ok {
		restY = ly
	}
	mb := drop.NewModelBuilder(t, f).
		SetItem(c.Body.ItemId, c.Body.Quantity).
		SetMeso(c.Body.Mesos).
		SetType(c.Body.DropType).
		SetPosition(restX, restY).
		SetOwner(c.Body.OwnerId, c.Body.OwnerPartyId).
		SetDropper(c.Body.DropperId, c.Body.DropperX, c.Body.DropperY).
		SetPlayerDrop(c.Body.PlayerDrop).
		SetStrength(c.Body.Strength).
		SetDexterity(c.Body.Dexterity).
		SetIntelligence(c.Body.Intelligence).
		SetLuck(c.Body.Luck).
		SetHp(c.Body.Hp).
		SetMp(c.Body.Mp).
		SetWeaponAttack(c.Body.WeaponAttack).
		SetMagicAttack(c.Body.MagicAttack).
		SetWeaponDefense(c.Body.WeaponDefense).
		SetMagicDefense(c.Body.MagicDefense).
		SetAccuracy(c.Body.Accuracy).
		SetAvoidability(c.Body.Avoidability).
		SetHands(c.Body.Hands).
		SetSpeed(c.Body.Speed).
		SetJump(c.Body.Jump).
		SetSlots(c.Body.Slots)
	p := drop.NewProcessor(l, ctx)
	_, _ = p.SpawnForCharacterAndEmit(mb)
}

func handleRequestReservation(l logrus.FieldLogger, ctx context.Context, c messageDropKafka.Command[messageDropKafka.CommandRequestReservationBody]) {
	if c.Type != messageDropKafka.CommandTypeRequestReservation {
		return
	}
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	_, _ = drop.NewProcessor(l, ctx).ReserveAndEmit(c.TransactionId, f, c.Body.DropId, c.Body.CharacterId, c.Body.PartyId, c.Body.PetSlot)
}

func handleCancelReservation(l logrus.FieldLogger, ctx context.Context, c messageDropKafka.Command[messageDropKafka.CommandCancelReservationBody]) {
	if c.Type != messageDropKafka.CommandTypeCancelReservation {
		return
	}
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	_ = drop.NewProcessor(l, ctx).CancelReservationAndEmit(c.TransactionId, f, c.Body.DropId, c.Body.CharacterId)
}

func handleRequestPickUp(l logrus.FieldLogger, ctx context.Context, c messageDropKafka.Command[messageDropKafka.CommandRequestPickUpBody]) {
	if c.Type != messageDropKafka.CommandTypeRequestPickUp {
		return
	}
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	_, _ = drop.NewProcessor(l, ctx).GatherAndEmit(c.TransactionId, f, c.Body.DropId, c.Body.CharacterId)
}

func handleConsume(l logrus.FieldLogger, ctx context.Context, c messageDropKafka.Command[messageDropKafka.CommandConsumeBody]) {
	if c.Type != messageDropKafka.CommandTypeConsume {
		return
	}
	f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
	_ = drop.NewProcessor(l, ctx).ConsumeAndEmit(f, c.Body.DropId)
}
