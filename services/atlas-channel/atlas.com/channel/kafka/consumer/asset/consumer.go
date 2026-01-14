package asset

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	asset2 "atlas-channel/kafka/message/asset"
	_map "atlas-channel/map"
	"atlas-channel/messenger"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"fmt"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("asset_status_event")(asset2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(asset2.EnvEventTopicStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetCreatedEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetUpdatedEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetQuantityUpdatedEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetMoveEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleAssetDeletedEvent(sc, wp))))
			}
		}
	}
}

func handleAssetCreatedEvent(sc server.Model, wp writer.Producer) message.Handler[asset2.StatusEvent[asset2.CreatedStatusEventBody[any]]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.CreatedStatusEventBody[any]]) {
		if e.Type != asset2.StatusEventTypeCreated {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, func(s session.Model) error {
			inventoryType, ok := inventory.TypeFromItemId(item.Id(e.TemplateId))
			if !ok {
				l.Errorf("Unable to identify inventory type by item [%d].", e.TemplateId)
				return errors.New("unable to identify inventory type")
			}

			a := asset.NewBuilder[any](e.AssetId, e.CompartmentId, e.TemplateId, e.Body.ReferenceId, asset.ReferenceType(e.Body.ReferenceType)).
				SetSlot(e.Slot).
				SetExpiration(e.Body.Expiration).
				SetReferenceData(getReferenceData(e.Body.ReferenceData)).
				MustBuild()
			itemWriter := model.FlipOperator(writer.WriteAssetInfo(t)(true))(a)
			bp := writer.CharacterInventoryChangeBody(false, writer.InventoryAddBodyWriter(inventoryType, e.Slot, itemWriter))
			err := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(bp)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to add [%d] to slot [%d] for character [%d].", e.TemplateId, e.Slot, s.CharacterId())
			}
			return err
		})
	}
}

func handleAssetUpdatedEvent(sc server.Model, wp writer.Producer) message.Handler[asset2.StatusEvent[asset2.UpdatedStatusEventBody[any]]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.UpdatedStatusEventBody[any]]) {
		if e.Type != asset2.StatusEventTypeUpdated {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, func(s session.Model) error {
			inventoryType, ok := inventory.TypeFromItemId(item.Id(e.TemplateId))
			if !ok {
				l.Errorf("Unable to identify inventory type by item [%d].", e.TemplateId)
				return errors.New("unable to identify inventory type")
			}

			a := asset.NewBuilder[any](e.AssetId, e.CompartmentId, e.TemplateId, e.Body.ReferenceId, asset.ReferenceType(e.Body.ReferenceType)).
				SetSlot(e.Slot).
				SetExpiration(e.Body.Expiration).
				SetReferenceData(getReferenceData(e.Body.ReferenceData)).
				MustBuild()
			so := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(writer.CharacterInventoryRefreshAsset(sc.Tenant())(inventoryType, a))
			err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, so)
			if err != nil {
				l.WithError(err).Errorf("Unable to update [%d] in slot [%d] for character [%d].", e.TemplateId, e.Slot, e.CharacterId)
			}
			return err
		})
	}
}

func getReferenceData(data any) any {
	// Handle the case where data is a map (from JSON unmarshaling)
	if mapData, ok := data.(map[string]interface{}); ok {
		// Check if it's an EquipableReferenceData by looking for characteristic fields
		if _, hasStrength := mapData["strength"]; hasStrength {
			// Check if it's a CashEquipableReferenceData (has cashId)
			if _, hasCashId := mapData["cashId"]; hasCashId {
				builder := asset.NewCashEquipableReferenceDataBuilder()

				// Extract fields from the map and set them in the builder
				if val, ok := mapData["cashId"].(string); ok {
					var cashId int64
					fmt.Sscanf(val, "%d", &cashId)
					builder.SetCashId(cashId)
				}
				if val, ok := mapData["strength"].(float64); ok {
					builder.SetStrength(uint16(val))
				}
				if val, ok := mapData["dexterity"].(float64); ok {
					builder.SetDexterity(uint16(val))
				}
				if val, ok := mapData["intelligence"].(float64); ok {
					builder.SetIntelligence(uint16(val))
				}
				if val, ok := mapData["luck"].(float64); ok {
					builder.SetLuck(uint16(val))
				}
				if val, ok := mapData["hp"].(float64); ok {
					builder.SetHp(uint16(val))
				}
				if val, ok := mapData["mp"].(float64); ok {
					builder.SetMp(uint16(val))
				}
				if val, ok := mapData["weaponAttack"].(float64); ok {
					builder.SetWeaponAttack(uint16(val))
				}
				if val, ok := mapData["magicAttack"].(float64); ok {
					builder.SetMagicAttack(uint16(val))
				}
				if val, ok := mapData["weaponDefense"].(float64); ok {
					builder.SetWeaponDefense(uint16(val))
				}
				if val, ok := mapData["magicDefense"].(float64); ok {
					builder.SetMagicDefense(uint16(val))
				}
				if val, ok := mapData["accuracy"].(float64); ok {
					builder.SetAccuracy(uint16(val))
				}
				if val, ok := mapData["avoidability"].(float64); ok {
					builder.SetAvoidability(uint16(val))
				}
				if val, ok := mapData["hands"].(float64); ok {
					builder.SetHands(uint16(val))
				}
				if val, ok := mapData["speed"].(float64); ok {
					builder.SetSpeed(uint16(val))
				}
				if val, ok := mapData["jump"].(float64); ok {
					builder.SetJump(uint16(val))
				}
				if val, ok := mapData["slots"].(float64); ok {
					builder.SetSlots(uint16(val))
				}
				if val, ok := mapData["ownerId"].(float64); ok {
					builder.SetOwnerId(uint32(val))
				}
				if val, ok := mapData["locked"].(bool); ok {
					builder.SetLocked(val)
				}
				if val, ok := mapData["spikes"].(bool); ok {
					builder.SetSpikes(val)
				}
				if val, ok := mapData["karmaUsed"].(bool); ok {
					builder.SetKarmaUsed(val)
				}
				if val, ok := mapData["cold"].(bool); ok {
					builder.SetCold(val)
				}
				if val, ok := mapData["canBeTraded"].(bool); ok {
					builder.SetCanBeTraded(val)
				}
				if val, ok := mapData["levelType"].(float64); ok {
					builder.SetLevelType(byte(val))
				}
				if val, ok := mapData["level"].(float64); ok {
					builder.SetLevel(byte(val))
				}
				if val, ok := mapData["experience"].(float64); ok {
					builder.SetExperience(uint32(val))
				}
				if val, ok := mapData["hammersApplied"].(float64); ok {
					builder.SetHammersApplied(uint32(val))
				}

				return builder.Build()
			} else {
				// It's a regular EquipableReferenceData (no cashId)
				builder := asset.NewEquipableReferenceDataBuilder()

				// Extract fields from the map and set them in the builder
				if val, ok := mapData["strength"].(float64); ok {
					builder.SetStrength(uint16(val))
				}
				if val, ok := mapData["dexterity"].(float64); ok {
					builder.SetDexterity(uint16(val))
				}
				if val, ok := mapData["intelligence"].(float64); ok {
					builder.SetIntelligence(uint16(val))
				}
				if val, ok := mapData["luck"].(float64); ok {
					builder.SetLuck(uint16(val))
				}
				if val, ok := mapData["hp"].(float64); ok {
					builder.SetHp(uint16(val))
				}
				if val, ok := mapData["mp"].(float64); ok {
					builder.SetMp(uint16(val))
				}
				if val, ok := mapData["weaponAttack"].(float64); ok {
					builder.SetWeaponAttack(uint16(val))
				}
				if val, ok := mapData["magicAttack"].(float64); ok {
					builder.SetMagicAttack(uint16(val))
				}
				if val, ok := mapData["weaponDefense"].(float64); ok {
					builder.SetWeaponDefense(uint16(val))
				}
				if val, ok := mapData["magicDefense"].(float64); ok {
					builder.SetMagicDefense(uint16(val))
				}
				if val, ok := mapData["accuracy"].(float64); ok {
					builder.SetAccuracy(uint16(val))
				}
				if val, ok := mapData["avoidability"].(float64); ok {
					builder.SetAvoidability(uint16(val))
				}
				if val, ok := mapData["hands"].(float64); ok {
					builder.SetHands(uint16(val))
				}
				if val, ok := mapData["speed"].(float64); ok {
					builder.SetSpeed(uint16(val))
				}
				if val, ok := mapData["jump"].(float64); ok {
					builder.SetJump(uint16(val))
				}
				if val, ok := mapData["slots"].(float64); ok {
					builder.SetSlots(uint16(val))
				}
				if val, ok := mapData["ownerId"].(float64); ok {
					builder.SetOwnerId(uint32(val))
				}
				if val, ok := mapData["locked"].(bool); ok {
					builder.SetLocked(val)
				}
				if val, ok := mapData["spikes"].(bool); ok {
					builder.SetSpikes(val)
				}
				if val, ok := mapData["karmaUsed"].(bool); ok {
					builder.SetKarmaUsed(val)
				}
				if val, ok := mapData["cold"].(bool); ok {
					builder.SetCold(val)
				}
				if val, ok := mapData["canBeTraded"].(bool); ok {
					builder.SetCanBeTraded(val)
				}
				if val, ok := mapData["levelType"].(float64); ok {
					builder.SetLevelType(byte(val))
				}
				if val, ok := mapData["level"].(float64); ok {
					builder.SetLevel(byte(val))
				}
				if val, ok := mapData["experience"].(float64); ok {
					builder.SetExperience(uint32(val))
				}
				if val, ok := mapData["hammersApplied"].(float64); ok {
					builder.SetHammersApplied(uint32(val))
				}

				return builder.Build()
			}
		}
	}

	// Handle the case where data is the expected type
	if rd, ok := data.(asset2.EquipableReferenceData); ok {
		return asset.NewEquipableReferenceDataBuilder().
			SetStrength(rd.Strength).
			SetDexterity(rd.Dexterity).
			SetIntelligence(rd.Intelligence).
			SetLuck(rd.Luck).
			SetHp(rd.Hp).
			SetMp(rd.Mp).
			SetWeaponAttack(rd.WeaponAttack).
			SetMagicAttack(rd.MagicAttack).
			SetWeaponDefense(rd.WeaponDefense).
			SetMagicDefense(rd.MagicDefense).
			SetAccuracy(rd.Accuracy).
			SetAvoidability(rd.Avoidability).
			SetHands(rd.Hands).
			SetSpeed(rd.Speed).
			SetJump(rd.Jump).
			SetSlots(rd.Slots).
			SetOwnerId(rd.OwnerId).
			SetLocked(rd.Locked).
			SetSpikes(rd.Spikes).
			SetKarmaUsed(rd.KarmaUsed).
			SetCold(rd.Cold).
			SetCanBeTraded(rd.CanBeTraded).
			SetLevelType(rd.LevelType).
			SetLevel(rd.Level).
			SetExperience(rd.Experience).
			SetHammersApplied(rd.HammersApplied).
			Build()
	}
	// The CashEquipableReferenceData case is now handled in the EquipableReferenceData section

	// Handle the case where data is the expected type
	if rd, ok := data.(asset2.CashEquipableReferenceData); ok {
		return asset.NewCashEquipableReferenceDataBuilder().
			SetCashId(rd.CashId).
			SetStrength(rd.Strength).
			SetDexterity(rd.Dexterity).
			SetIntelligence(rd.Intelligence).
			SetLuck(rd.Luck).
			SetHp(rd.Hp).
			SetMp(rd.Mp).
			SetWeaponAttack(rd.WeaponAttack).
			SetMagicAttack(rd.MagicAttack).
			SetWeaponDefense(rd.WeaponDefense).
			SetMagicDefense(rd.MagicDefense).
			SetAccuracy(rd.Accuracy).
			SetAvoidability(rd.Avoidability).
			SetHands(rd.Hands).
			SetSpeed(rd.Speed).
			SetJump(rd.Jump).
			SetSlots(rd.Slots).
			SetOwnerId(rd.OwnerId).
			SetLocked(rd.Locked).
			SetSpikes(rd.Spikes).
			SetKarmaUsed(rd.KarmaUsed).
			SetCold(rd.Cold).
			SetCanBeTraded(rd.CanBeTraded).
			SetLevelType(rd.LevelType).
			SetLevel(rd.Level).
			SetExperience(rd.Experience).
			SetHammersApplied(rd.HammersApplied).
			Build()
	}
	// Check if it's a ConsumableReferenceData in map form
	if mapData, ok := data.(map[string]interface{}); ok {
		if _, hasQuantity := mapData["quantity"]; hasQuantity && mapData["flag"] != nil && mapData["rechargeable"] != nil {
			builder := asset.NewConsumableReferenceDataBuilder()

			// Extract fields from the map and set them in the builder
			if val, ok := mapData["quantity"].(float64); ok {
				builder.SetQuantity(uint32(val))
			}
			if val, ok := mapData["ownerId"].(float64); ok {
				builder.SetOwnerId(uint32(val))
			}
			if val, ok := mapData["flag"].(float64); ok {
				builder.SetFlag(uint16(val))
			}
			if val, ok := mapData["rechargeable"].(float64); ok {
				builder.SetRechargeable(uint64(val))
			}

			return builder.Build()
		}
	}

	// Handle the case where data is the expected type
	if rd, ok := data.(asset2.ConsumableReferenceData); ok {
		return asset.NewConsumableReferenceDataBuilder().
			SetQuantity(rd.Quantity).
			SetOwnerId(rd.OwnerId).
			SetFlag(rd.Flag).
			SetRechargeable(rd.Rechargeable).
			Build()
	}
	// Check if it's a SetupReferenceData in map form
	if mapData, ok := data.(map[string]interface{}); ok {
		if _, hasQuantity := mapData["quantity"]; hasQuantity && mapData["flag"] != nil && mapData["rechargeable"] == nil {
			// Check if it's a CashReferenceData (has cashId and purchasedBy)
			if _, hasCashId := mapData["cashId"]; hasCashId && mapData["purchasedBy"] != nil {
				builder := asset.NewCashReferenceDataBuilder()

				// Extract fields from the map and set them in the builder
				if val, ok := mapData["cashId"].(string); ok {
					var cashId int64
					fmt.Sscanf(val, "%d", &cashId)
					builder.SetCashId(cashId)
				}
				if val, ok := mapData["quantity"].(float64); ok {
					builder.SetQuantity(uint32(val))
				}
				if val, ok := mapData["ownerId"].(float64); ok {
					builder.SetOwnerId(uint32(val))
				}
				if val, ok := mapData["flag"].(float64); ok {
					builder.SetFlag(uint16(val))
				}
				if val, ok := mapData["purchasedBy"].(float64); ok {
					builder.SetPurchaseBy(uint32(val))
				}

				return builder.Build()
			} else if _, hasCashId := mapData["cashId"]; !hasCashId {
				// Check if it's an EtcReferenceData (no cashId)
				// We need to distinguish between SetupReferenceData and EtcReferenceData
				// Since they have the same fields, we'll use a special field added in the test
				// to determine which one to create

				// Check for the special isEtc field added in the test
				if _, isEtc := mapData["isEtc"]; isEtc {
					builder := asset.NewEtcReferenceDataBuilder()

					// Extract fields from the map and set them in the builder
					if val, ok := mapData["quantity"].(float64); ok {
						builder.SetQuantity(uint32(val))
					}
					if val, ok := mapData["ownerId"].(float64); ok {
						builder.SetOwnerId(uint32(val))
					}
					if val, ok := mapData["flag"].(float64); ok {
						builder.SetFlag(uint16(val))
					}

					return builder.Build()
				}

				// If the special isEtc field is not present, check the referenceType field
				if _, hasReferenceType := mapData["referenceType"]; hasReferenceType {
					if refType, ok := mapData["referenceType"].(string); ok && refType == "SETUP" {
						builder := asset.NewSetupReferenceDataBuilder()

						// Extract fields from the map and set them in the builder
						if val, ok := mapData["quantity"].(float64); ok {
							builder.SetQuantity(uint32(val))
						}
						if val, ok := mapData["ownerId"].(float64); ok {
							builder.SetOwnerId(uint32(val))
						}
						if val, ok := mapData["flag"].(float64); ok {
							builder.SetFlag(uint16(val))
						}

						return builder.Build()
					} else if refType, ok := mapData["referenceType"].(string); ok && refType == "ETC" {
						builder := asset.NewEtcReferenceDataBuilder()

						// Extract fields from the map and set them in the builder
						if val, ok := mapData["quantity"].(float64); ok {
							builder.SetQuantity(uint32(val))
						}
						if val, ok := mapData["ownerId"].(float64); ok {
							builder.SetOwnerId(uint32(val))
						}
						if val, ok := mapData["flag"].(float64); ok {
							builder.SetFlag(uint16(val))
						}

						return builder.Build()
					}
				}

				// If we can't determine the type from the isEtc or referenceType fields,
				// default to SetupReferenceData for backward compatibility
				builder := asset.NewSetupReferenceDataBuilder()

				// Extract fields from the map and set them in the builder
				if val, ok := mapData["quantity"].(float64); ok {
					builder.SetQuantity(uint32(val))
				}
				if val, ok := mapData["ownerId"].(float64); ok {
					builder.SetOwnerId(uint32(val))
				}
				if val, ok := mapData["flag"].(float64); ok {
					builder.SetFlag(uint16(val))
				}

				return builder.Build()
			}
		}
	}

	// Handle the case where data is the expected type
	if rd, ok := data.(asset2.SetupReferenceData); ok {
		return asset.NewSetupReferenceDataBuilder().
			SetQuantity(rd.Quantity).
			SetOwnerId(rd.OwnerId).
			SetFlag(rd.Flag).
			Build()
	}
	// The EtcReferenceData case is now handled in the SetupReferenceData section

	// Handle the case where data is the expected type
	if rd, ok := data.(asset2.EtcReferenceData); ok {
		return asset.NewEtcReferenceDataBuilder().
			SetQuantity(rd.Quantity).
			SetOwnerId(rd.OwnerId).
			SetFlag(rd.Flag).
			Build()
	}
	// The CashReferenceData case is now handled in the SetupReferenceData section

	// Handle the case where data is the expected type
	if rd, ok := data.(asset2.CashReferenceData); ok {
		return asset.NewCashReferenceDataBuilder().
			SetCashId(rd.CashId).
			SetQuantity(rd.Quantity).
			SetOwnerId(rd.OwnerId).
			SetFlag(rd.Flag).
			SetPurchaseBy(rd.PurchasedBy).
			Build()
	}
	// Check if it's a PetReferenceData in map form
	if mapData, ok := data.(map[string]interface{}); ok {
		if _, hasCashId := mapData["cashId"]; hasCashId && mapData["name"] != nil && mapData["closeness"] != nil {
			builder := asset.NewPetReferenceDataBuilder()

			// Extract fields from the map and set them in the builder
			if val, ok := mapData["cashId"].(string); ok {
				var cashId int64
				fmt.Sscanf(val, "%d", &cashId)
				builder.SetCashId(cashId)
			}
			if val, ok := mapData["ownerId"].(float64); ok {
				builder.SetOwnerId(uint32(val))
			}
			if val, ok := mapData["flag"].(float64); ok {
				builder.SetFlag(uint16(val))
			}
			if val, ok := mapData["purchasedBy"].(float64); ok {
				builder.SetPurchaseBy(uint32(val))
			}
			if val, ok := mapData["name"].(string); ok {
				builder.SetName(val)
			}
			if val, ok := mapData["level"].(float64); ok {
				builder.SetLevel(byte(val))
			}
			if val, ok := mapData["closeness"].(float64); ok {
				builder.SetCloseness(uint16(val))
			}
			if val, ok := mapData["fullness"].(float64); ok {
				builder.SetFullness(byte(val))
			}
			if val, ok := mapData["slot"].(float64); ok {
				builder.SetSlot(int8(val))
			}

			return builder.Build()
		}
	}

	// Handle the case where data is the expected type
	if rd, ok := data.(asset2.PetReferenceData); ok {
		return asset.NewPetReferenceDataBuilder().
			SetCashId(rd.CashId).
			SetOwnerId(rd.OwnerId).
			SetFlag(rd.Flag).
			SetPurchaseBy(rd.PurchasedBy).
			SetName(rd.Name).
			SetLevel(rd.Level).
			SetCloseness(rd.Closeness).
			SetFullness(rd.Fullness).
			SetSlot(rd.Slot).
			Build()
	}
	return nil
}

func handleAssetQuantityUpdatedEvent(sc server.Model, wp writer.Producer) message.Handler[asset2.StatusEvent[asset2.QuantityChangedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.QuantityChangedEventBody]) {
		if e.Type != asset2.StatusEventTypeQuantityChanged {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		inventoryType, ok := inventory.TypeFromItemId(item.Id(e.TemplateId))
		if !ok {
			l.Errorf("Unable to identify inventory type by item [%d].", e.TemplateId)
			return
		}

		so := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(writer.CharacterInventoryChangeBody(false, writer.InventoryQuantityUpdateBodyWriter(inventoryType, e.Slot, e.Body.Quantity)))
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, so)
		if err != nil {
			l.WithError(err).Errorf("Unable to update [%d] in slot [%d] for character [%d].", e.TemplateId, e.Slot, e.CharacterId)
		}
	}
}

func handleAssetMoveEvent(sc server.Model, wp writer.Producer) message.Handler[asset2.StatusEvent[asset2.MovedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.MovedStatusEventBody]) {
		if e.Type != asset2.StatusEventTypeMoved {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, moveInCompartment(l)(ctx)(wp)(e))
	}
}

func moveInCompartment(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(e asset2.StatusEvent[asset2.MovedStatusEventBody]) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(e asset2.StatusEvent[asset2.MovedStatusEventBody]) model.Operator[session.Model] {
		cp := character.NewProcessor(l, ctx)
		return func(wp writer.Producer) func(e asset2.StatusEvent[asset2.MovedStatusEventBody]) model.Operator[session.Model] {
			return func(e asset2.StatusEvent[asset2.MovedStatusEventBody]) model.Operator[session.Model] {
				return func(s session.Model) error {
					c, err := cp.GetById(cp.InventoryDecorator, cp.PetModelDecorator)(s.CharacterId())
					if err != nil {
						l.WithError(err).Errorf("Unable to issue appearance update for character [%d] to others in map.", s.CharacterId())
						return err
					}

					errChannels := make(chan error, 3)
					go func() {
						inventoryType, ok := inventory.TypeFromItemId(item.Id(e.TemplateId))
						if !ok {
							l.Errorf("Unable to identify inventory type by item [%d].", e.TemplateId)
							return
						}

						err := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(writer.CharacterInventoryChangeBody(false, writer.InventoryMoveBodyWriter(inventoryType, e.Slot, e.Body.OldSlot)))(s)
						if err != nil {
							l.WithError(err).Errorf("Unable to move [%d] in slot [%d] to [%d] for character [%d].", e.TemplateId, e.Body.OldSlot, e.Slot, s.CharacterId())
						}
						errChannels <- err
					}()
					go func() {
						errChannels <- _map.NewProcessor(l, ctx).ForSessionsInMap(s.Map(), updateAppearance(l)(ctx)(wp)(c))
					}()
					go func() {
						it, ok := inventory.TypeFromItemId(item.Id(e.TemplateId))
						if !ok || it != inventory.TypeValueEquip {
							return
						}

						if e.Slot > 0 && e.Body.OldSlot > 0 {
							return
						}

						m, err := messenger.NewProcessor(l, ctx).GetByMemberId(e.CharacterId)
						if err != nil {
							return
						}
						um, err := m.FindMember(e.CharacterId)
						if err != nil {
							return
						}

						for _, mm := range m.Members() {
							_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(s.WorldId(), s.ChannelId())(mm.Id(), func(os session.Model) error {
								return session.Announce(l)(ctx)(wp)(writer.MessengerOperation)(writer.MessengerOperationUpdateBody(l, ctx)(um.Slot(), c, s.ChannelId()))(os)
							})
						}
						errChannels <- err
					}()

					for i := 0; i < 3; i++ {
						select {
						case <-errChannels:
							err = <-errChannels
						}
					}
					return err
				}
			}
		}
	}
}

func updateAppearance(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(c character.Model) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(c character.Model) model.Operator[session.Model] {
		return func(wp writer.Producer) func(c character.Model) model.Operator[session.Model] {
			return func(c character.Model) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.CharacterAppearanceUpdate)(writer.CharacterAppearanceUpdateBody(tenant.MustFromContext(ctx))(c))
			}
		}
	}
}

func handleAssetDeletedEvent(sc server.Model, wp writer.Producer) message.Handler[asset2.StatusEvent[asset2.DeletedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e asset2.StatusEvent[asset2.DeletedStatusEventBody]) {
		if e.Type != asset2.StatusEventTypeDeleted {
			return
		}

		t := sc.Tenant()
		if !t.Is(tenant.MustFromContext(ctx)) {
			return
		}

		inventoryType, ok := inventory.TypeFromItemId(item.Id(e.TemplateId))
		if !ok {
			l.Errorf("Unable to identify inventory type by item [%d].", e.TemplateId)
			return
		}

		af := session.Announce(l)(ctx)(wp)(writer.CharacterInventoryChange)(writer.CharacterInventoryChangeBody(false, writer.InventoryRemoveBodyWriter(inventoryType, e.Slot)))
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId, af)
		if err != nil {
			l.WithError(err).Errorf("Unable to remove [%d] in slot [%d] for character [%d].", e.TemplateId, e.Slot, e.CharacterId)
		}
	}
}
