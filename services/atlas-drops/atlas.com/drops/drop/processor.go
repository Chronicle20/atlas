package drop

import (
	"atlas-drops/equipment"
	"atlas-drops/inventory"
	"atlas-drops/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

func Spawn(l logrus.FieldLogger) func(ctx context.Context) func(mb *ModelBuilder) error {
	return func(ctx context.Context) func(mb *ModelBuilder) error {
		return func(mb *ModelBuilder) error {
			it, _ := inventory.GetInventoryType(mb.ItemId())
			if it == inventory.TypeValueEquip {
				e, err := equipment.Create(l)(ctx)(mb.ItemId())()
				if err != nil {
					l.WithError(err).Errorf("Unable to generate [%d] equipment for drop.", mb.ItemId())
					return err
				}

				mb.SetEquipmentId(e.Id())
			}
			m := GetRegistry().CreateDrop(mb)
			_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicDropStatus)(createdEventStatusProvider(m))
			return nil
		}
	}
}

func SpawnForCharacter(l logrus.FieldLogger) func(ctx context.Context) func(mb *ModelBuilder) error {
	return func(ctx context.Context) func(mb *ModelBuilder) error {
		return func(mb *ModelBuilder) error {
			m := GetRegistry().CreateDrop(mb)
			_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicDropStatus)(createdEventStatusProvider(m))
			return nil
		}
	}
}

func Reserve(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32, petSlot int8) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32, petSlot int8) error {
		return func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32, petSlot int8) error {
			d, err := GetRegistry().ReserveDrop(dropId, characterId, petSlot)
			if err == nil {
				l.Debugf("Reserving [%d] for [%d].", dropId, characterId)
				_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicDropStatus)(reservedEventStatusProvider(worldId, channelId, mapId, dropId, characterId, d.ItemId(), d.EquipmentId(), d.Quantity(), d.Meso()))
			} else {
				l.Debugf("Failed reserving [%d] for [%d].", dropId, characterId)
				_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicDropStatus)(reservationFailureEventStatusProvider(worldId, channelId, mapId, dropId, characterId))
			}
			return err
		}
	}
}

func CancelReservation(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) error {
		return func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) error {
			GetRegistry().CancelDropReservation(dropId, characterId)
			_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicDropStatus)(reservationFailureEventStatusProvider(worldId, channelId, mapId, dropId, characterId))
			return nil
		}
	}
}

func Gather(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) error {
		return func(worldId byte, channelId byte, mapId uint32, dropId uint32, characterId uint32) error {
			d, err := GetRegistry().RemoveDrop(dropId)
			if d.Id() == 0 || err == nil {
				l.Debugf("Gathering [%d] for [%d].", dropId, characterId)
				_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicDropStatus)(pickedUpEventStatusProvider(worldId, channelId, mapId, dropId, characterId, d.ItemId(), d.EquipmentId(), d.Quantity(), d.Meso(), d.PetSlot()))
			}
			return err
		}
	}
}

func GetById(l logrus.FieldLogger) func(ctx context.Context) func(dropId uint32) (Model, error) {
	return func(ctx context.Context) func(dropId uint32) (Model, error) {
		return func(dropId uint32) (Model, error) {
			return GetRegistry().GetDrop(dropId)
		}
	}
}

func GetForMap(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) ([]Model, error) {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32) ([]Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(worldId byte, channelId byte, mapId uint32) ([]Model, error) {
			return GetRegistry().GetDropsForMap(t, worldId, channelId, mapId)
		}
	}
}

var AllProvider = model.FixedProvider(GetRegistry().GetAllDrops())

func Expire(l logrus.FieldLogger) func(ctx context.Context) model.Operator[Model] {
	return func(ctx context.Context) model.Operator[Model] {
		return func(m Model) error {
			_, err := GetRegistry().RemoveDrop(m.Id())
			if err != nil {
				l.WithError(err).Errorf("Unable to remove drop [%d] from registry.", m.Id())
				return err
			}

			if m.EquipmentId() != 0 {
				err = equipment.Delete(l)(ctx)(m.EquipmentId())
				if err != nil {
					l.WithError(err).Errorf("Unable to delete equipment [%d] corresponding to drop [%d].", m.EquipmentId(), m.Id())
					return err
				}
			}

			_ = producer.ProviderImpl(l)(ctx)(EnvEventTopicDropStatus)(expiredEventStatusProvider(m.WorldId(), m.ChannelId(), m.MapId(), m.Id()))
			return nil
		}
	}
}
