package character

import (
	"atlas-character/database"
	"atlas-character/equipable"
	"atlas-character/equipment"
	"atlas-character/equipment/slot"
	"atlas-character/inventory"
	"atlas-character/kafka/producer"
	"atlas-character/portal"
	"atlas-character/tenant"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"regexp"
)

var blockedNameErr = errors.New("blocked name")
var invalidLevelErr = errors.New("invalid level")

func byIdProvider(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(characterId uint32) model.Provider[Model] {
	return func(characterId uint32) model.Provider[Model] {
		return database.ModelProvider[Model, entity](db)(getById(tenant.Id, characterId), makeCharacter)
	}
}

func GetById(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(characterId uint32, decorators ...model.Decorator[Model]) (Model, error) {
	return func(characterId uint32, decorators ...model.Decorator[Model]) (Model, error) {
		return model.Map(byIdProvider(l, db, tenant)(characterId), model.Decorate(decorators...))()
	}
}

func byAccountInWorldProvider(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(accountId uint32, worldId byte) model.Provider[[]Model] {
	return func(accountId uint32, worldId byte) model.Provider[[]Model] {
		return database.ModelSliceProvider[Model, entity](db)(getForAccountInWorld(tenant.Id, accountId, worldId), makeCharacter)
	}
}

func GetForAccountInWorld(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(accountId uint32, worldId byte, decorators ...model.Decorator[Model]) ([]Model, error) {
	return func(accountId uint32, worldId byte, decorators ...model.Decorator[Model]) ([]Model, error) {
		return model.SliceMap(byAccountInWorldProvider(l, db, tenant)(accountId, worldId), model.Decorate(decorators...))()
	}
}

func byMapInWorld(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(worldId byte, mapId uint32) model.Provider[[]Model] {
	return func(worldId byte, mapId uint32) model.Provider[[]Model] {
		return database.ModelSliceProvider[Model, entity](db)(getForMapInWorld(tenant.Id, worldId, mapId), makeCharacter)
	}
}

func GetForMapInWorld(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(worldId byte, mapId uint32, decorators ...model.Decorator[Model]) ([]Model, error) {
	return func(worldId byte, mapId uint32, decorators ...model.Decorator[Model]) ([]Model, error) {
		return model.SliceMap(byMapInWorld(l, db, tenant)(worldId, mapId), model.Decorate(decorators...))()
	}
}

func byName(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(name string) model.Provider[[]Model] {
	return func(name string) model.Provider[[]Model] {
		return database.ModelSliceProvider[Model, entity](db)(getForName(tenant.Id, name), makeCharacter)
	}
}

func GetForName(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(name string, decorators ...model.Decorator[Model]) ([]Model, error) {
	return func(name string, decorators ...model.Decorator[Model]) ([]Model, error) {
		return model.SliceMap(byName(l, db, tenant)(name), model.Decorate(decorators...))()
	}
}

func InventoryModelDecorator(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) model.Decorator[Model] {
	return func(m Model) Model {
		i, err := inventory.GetInventories(l, db, span, tenant)(m.Id())
		if err != nil {
			return m
		}

		es, err := model.Fold(equipable.EquipmentProvider(l, db, span, tenant)(i.Equipable().Id()), model.FixedProvider(m.GetEquipment()), FoldEquipable)()
		if err != nil {
			return CloneModel(m).SetInventory(i).Build()
		}

		return CloneModel(m).SetEquipment(es).SetInventory(i).Build()
	}
}

func FoldEquipable(m equipment.Model, e equipable.Model) (equipment.Model, error) {
	var setter equipment.SlotSetter
	if e.Slot() > -100 {
		switch slot.Position(e.Slot()) {
		case slot.PositionHat:
			setter = m.SetHat
		case slot.PositionMedal:
			setter = m.SetMedal
		case slot.PositionForehead:
			setter = m.SetForehead
		case slot.PositionRing1:
			setter = m.SetRing1
		case slot.PositionRing2:
			setter = m.SetRing2
		case slot.PositionEye:
			setter = m.SetEye
		case slot.PositionEarring:
			setter = m.SetEarring
		case slot.PositionShoulder:
			setter = m.SetShoulder
		case slot.PositionCape:
			setter = m.SetCape
		case slot.PositionTop:
			setter = m.SetTop
		case slot.PositionPendant:
			setter = m.SetPendant
		case slot.PositionWeapon:
			setter = m.SetWeapon
		case slot.PositionShield:
			setter = m.SetShield
		case slot.PositionGloves:
			setter = m.SetGloves
		case slot.PositionBottom:
			setter = m.SetBottom
		case slot.PositionBelt:
			setter = m.SetBelt
		case slot.PositionRing3:
			setter = m.SetRing3
		case slot.PositionRing4:
			setter = m.SetRing4
		case slot.PositionShoes:
			setter = m.SetShoes
		}
	} else {
		switch slot.Position(e.Slot() + 100) {
		case slot.PositionHat:
			setter = m.SetCashHat
		case slot.PositionMedal:
			setter = m.SetCashMedal
		case slot.PositionForehead:
			setter = m.SetCashForehead
		case slot.PositionRing1:
			setter = m.SetCashRing1
		case slot.PositionRing2:
			setter = m.SetCashRing2
		case slot.PositionEye:
			setter = m.SetCashEye
		case slot.PositionEarring:
			setter = m.SetCashEarring
		case slot.PositionShoulder:
			setter = m.SetCashShoulder
		case slot.PositionCape:
			setter = m.SetCashCape
		case slot.PositionTop:
			setter = m.SetCashTop
		case slot.PositionPendant:
			setter = m.SetCashPendant
		case slot.PositionWeapon:
			setter = m.SetCashWeapon
		case slot.PositionShield:
			setter = m.SetCashShield
		case slot.PositionGloves:
			setter = m.SetCashGloves
		case slot.PositionBottom:
			setter = m.SetCashBottom
		case slot.PositionBelt:
			setter = m.SetCashBelt
		case slot.PositionRing3:
			setter = m.SetCashRing3
		case slot.PositionRing4:
			setter = m.SetCashRing4
		case slot.PositionShoes:
			setter = m.SetCashShoes
		}
	}
	return setter(&e), nil
}

func IsValidName(l logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(name string) (bool, error) {
	return func(name string) (bool, error) {
		m, err := regexp.MatchString("[A-Za-z0-9\u3040-\u309F\u30A0-\u30FF\u4E00-\u9FAF]{3,12}", name)
		if err != nil {
			return false, err
		}
		if !m {
			return false, nil
		}

		cs, err := GetForName(l, db, tenant)(name)
		if len(cs) != 0 || err != nil {
			return false, nil
		}

		//TODO
		//bn, err := blocked_name.IsBlockedName(l, span)(name)
		//if bn {
		//	return false, err
		//}

		return true, nil
	}
}

func Create(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, eventProducer producer.Provider) func(tenant tenant.Model, input Model) (Model, error) {
	return func(tenant tenant.Model, input Model) (Model, error) {
		ok, err := IsValidName(l, db, tenant)(input.Name())
		if err != nil {
			l.WithError(err).Errorf("Error validating name [%s] during character creation.", input.Name())
			return Model{}, err
		}
		if !ok {
			l.Infof("Attempting to create a character with an invalid name [%s].", input.Name())
			return Model{}, blockedNameErr
		}
		if input.Level() < 1 || input.Level() > 200 {
			l.Infof("Attempting to create character with an invalid level [%d].", input.Level())
			return Model{}, invalidLevelErr
		}

		var res Model
		err = db.Transaction(func(tx *gorm.DB) error {
			res, err = create(tx, tenant.Id, input.accountId, input.worldId, input.name, input.level, input.strength, input.dexterity, input.intelligence, input.luck, input.maxHp, input.maxMp, input.jobId, input.gender, input.hair, input.face, input.skinColor, input.mapId)
			if err != nil {
				l.WithError(err).Errorf("Error persisting character in database.")
				tx.Rollback()
				return err
			}

			inv, err := inventory.Create(l, tx, span, tenant)(res.id, 24)
			if err != nil {
				l.WithError(err).Errorf("Unable to create inventory for character during character creation.")
				tx.Rollback()
				return err
			}
			res = CloneModel(res).SetInventory(inv).Build()
			return nil
		})

		if err == nil {
			err = eventProducer(EnvEventTopicCharacterStatus)(createdEventProvider(tenant, res.Id(), res.WorldId(), res.Name()))
		}
		return res, err
	}
}

func Delete(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32) error {
	return func(characterId uint32) error {
		err := db.Transaction(func(tx *gorm.DB) error {
			c, err := GetById(l, tx, tenant)(characterId, InventoryModelDecorator(l, tx, span, tenant))
			if err != nil {
				return err
			}

			// delete equipment.
			err = equipment.Delete(l, tx, span, tenant)(c.equipment)
			if err != nil {
				l.WithError(err).Errorf("Unable to delete equipment for character with id [%d].", characterId)
				return err
			}

			// delete inventories.
			err = inventory.DeleteEquipableInventory(l, tx, span, tenant)(characterId, c.inventory.Equipable())
			if err != nil {
				l.WithError(err).Errorf("Unable to delete inventory for character with id [%d].", characterId)
				return err
			}
			err = inventory.DeleteItemInventory(l, tx, span, tenant)(characterId, c.inventory.Useable())
			if err != nil {
				l.WithError(err).Errorf("Unable to delete inventory for character with id [%d].", characterId)
				return err
			}
			err = inventory.DeleteItemInventory(l, tx, span, tenant)(characterId, c.inventory.Setup())
			if err != nil {
				l.WithError(err).Errorf("Unable to delete inventory for character with id [%d].", characterId)
				return err
			}
			err = inventory.DeleteItemInventory(l, tx, span, tenant)(characterId, c.inventory.Etc())
			if err != nil {
				l.WithError(err).Errorf("Unable to delete inventory for character with id [%d].", characterId)
				return err
			}
			err = inventory.DeleteItemInventory(l, tx, span, tenant)(characterId, c.inventory.Cash())
			if err != nil {
				l.WithError(err).Errorf("Unable to delete inventory for character with id [%d].", characterId)
				return err
			}

			_ = inventory.GetLockRegistry().DeleteForCharacter(characterId)

			err = delete(tx, tenant.Id, characterId)
			if err != nil {
				return err
			}

			return nil
		})
		return err
	}
}

func Login(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, worldId byte, channelId byte) {
	return func(characterId uint32, worldId byte, channelId byte) {
		c, err := GetById(l, db, tenant)(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate character [%d] whose session was created.", characterId)
			return
		}
		_ = producer.ProviderImpl(l)(span)(EnvEventTopicCharacterStatus)(loginEventProvider(tenant, characterId, worldId, channelId, c.MapId()))
	}
}

func Logout(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, worldId byte, channelId byte) {
	return func(characterId uint32, worldId byte, channelId byte) {
		c, err := GetById(l, db, tenant)(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate character [%d] whose session was destroyed.", characterId)
			return
		}
		_ = producer.ProviderImpl(l)(span)(EnvEventTopicCharacterStatus)(logoutEventProvider(tenant, characterId, worldId, channelId, c.MapId()))
	}
}

func ChangeMap(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(characterId uint32, worldId byte, channelId byte, mapId uint32, portalId uint32) {
	return func(characterId uint32, worldId byte, channelId byte, mapId uint32, portalId uint32) {
		c, err := GetById(l, db, tenant)(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to locate character [%d] for update.", characterId)
			return
		}
		err = performChangeMap(l, db, span, tenant)(mapId, portalId)(c)
		if err != nil {
			l.WithError(err).Errorf("Error updating characters [%d] map.", characterId)
			return
		}
		_ = changeMapSuccess(l, span, tenant)(worldId, channelId, c.MapId(), mapId, portalId)(c)
	}
}

func changeMapSuccess(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, oldMapId uint32, targetMapId uint32, targetPortalId uint32) model.Operator[Model] {
	return func(worldId byte, channelId byte, oldMapId uint32, targetMapId uint32, targetPortalId uint32) model.Operator[Model] {
		return func(m Model) error {
			return producer.ProviderImpl(l)(span)(EnvEventTopicCharacterStatus)(mapChangedEventProvider(tenant, m.Id(), worldId, channelId, oldMapId, targetMapId, targetPortalId))
		}
	}
}

// Produces a function which persists a character map update, then updates the temporal position.
func performChangeMap(l logrus.FieldLogger, db *gorm.DB, span opentracing.Span, tenant tenant.Model) func(mapId uint32, portalId uint32) model.Operator[Model] {
	return func(mapId uint32, portalId uint32) model.Operator[Model] {
		return func(c Model) error {
			err := characterDatabaseUpdate(l, db, tenant)(SetMapId(mapId))(c)
			if err != nil {
				return err
			}
			por, err := portal.GetInMapById(l, span, tenant)(mapId, portalId)
			if err != nil {
				return err
			}
			GetTemporalRegistry().UpdatePosition(c.Id(), por.X(), por.Y())
			return nil
		}
	}
}

// Returns a function which accepts a character model,and updates the persisted state of the character given a set of
// modifying functions.
func characterDatabaseUpdate(_ logrus.FieldLogger, db *gorm.DB, tenant tenant.Model) func(modifiers ...EntityUpdateFunction) model.Operator[Model] {
	return func(modifiers ...EntityUpdateFunction) model.Operator[Model] {
		return func(c Model) error {
			if len(modifiers) > 0 {
				err := update(db, tenant.Id, c.Id(), modifiers...)
				if err != nil {
					return err
				}
			}
			return nil
		}
	}
}

func Move(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32, worldId byte, channelId byte, mapId uint32, movement movement) {
	return func(characterId uint32, worldId byte, channelId byte, mapId uint32, movement movement) {
		var x = movement.StartX
		var y = movement.StartY
		var stance = GetTemporalRegistry().GetById(characterId).Stance()
		for _, m := range movement.Elements {
			if m.TypeStr == MovementTypeNormal {
				x = m.X
				y = m.Y
				stance = m.MoveAction
			} else if m.TypeStr == MovementTypeJump || m.TypeStr == MovementTypeTeleport || m.TypeStr == MovementTypeStartFallDown {
				stance = m.MoveAction
			}
		}
		GetTemporalRegistry().Update(characterId, x, y, stance)

		_ = producer.ProviderImpl(l)(span)(EnvEventTopicMovement)(move(tenant, worldId, channelId, mapId, characterId, movement))
	}
}
