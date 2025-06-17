package compartment

import (
	"atlas-inventory/asset"
	"atlas-inventory/data/equipment"
	"atlas-inventory/database"
	"atlas-inventory/drop"
	"atlas-inventory/kafka/message"
	"atlas-inventory/kafka/message/compartment"
	"atlas-inventory/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas-constants/item"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
	"math"
	"sort"
	"time"

	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor struct {
	l                  logrus.FieldLogger
	ctx                context.Context
	db                 *gorm.DB
	t                  tenant.Model
	assetProcessor     *asset.Processor
	dropProcessor      *drop.Processor
	equipmentProcessor *equipment.Processor
	producer           producer.Provider
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *Processor {
	p := &Processor{
		l:                  l,
		ctx:                ctx,
		db:                 db,
		t:                  tenant.MustFromContext(ctx),
		assetProcessor:     asset.NewProcessor(l, ctx, db),
		dropProcessor:      drop.NewProcessor(l, ctx),
		equipmentProcessor: equipment.NewProcessor(l, ctx),
		producer:           producer.ProviderImpl(l)(ctx),
	}
	return p
}

func (p *Processor) WithTransaction(db *gorm.DB) *Processor {
	return &Processor{
		l:                  p.l,
		ctx:                p.ctx,
		db:                 db,
		t:                  p.t,
		assetProcessor:     p.assetProcessor,
		dropProcessor:      p.dropProcessor,
		equipmentProcessor: p.equipmentProcessor,
		producer:           p.producer,
	}
}

func (p *Processor) WithAssetProcessor(ap *asset.Processor) *Processor {
	return &Processor{
		l:                  p.l,
		ctx:                p.ctx,
		db:                 p.db,
		t:                  p.t,
		assetProcessor:     ap,
		dropProcessor:      p.dropProcessor,
		equipmentProcessor: p.equipmentProcessor,
		producer:           p.producer,
	}
}

func (p *Processor) ByIdProvider(id uuid.UUID) model.Provider[Model] {
	cs, err := model.Map(Make)(getById(p.t.Id(), id)(p.db))()
	if err != nil {
		return model.ErrorProvider[Model](err)
	}
	return model.Map(p.DecorateAsset)(model.FixedProvider(cs))
}

func (p *Processor) GetById(id uuid.UUID) (Model, error) {
	return p.ByIdProvider(id)()
}

func (p *Processor) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	cs, err := model.SliceMap(Make)(getByCharacter(p.t.Id(), characterId)(p.db))(model.ParallelMap())()
	if err != nil {
		return model.ErrorProvider[[]Model](err)
	}
	return model.SliceMap(p.DecorateAsset)(model.FixedProvider(cs))(model.ParallelMap())
}

func (p *Processor) GetByCharacterId(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *Processor) ByCharacterAndTypeProvider(characterId uint32) func(inventoryType inventory.Type) model.Provider[Model] {
	return func(inventoryType inventory.Type) model.Provider[Model] {
		cs, err := model.Map(Make)(getByCharacterAndType(p.t.Id(), characterId, inventoryType)(p.db))()
		if err != nil {
			return model.ErrorProvider[Model](err)
		}
		return model.Map(p.DecorateAsset)(model.FixedProvider(cs))
	}
}

func (p *Processor) GetByCharacterAndType(characterId uint32) func(inventoryType inventory.Type) (Model, error) {
	return func(inventoryType inventory.Type) (Model, error) {
		return p.ByCharacterAndTypeProvider(characterId)(inventoryType)()
	}

}

func (p *Processor) DecorateAsset(m Model) (Model, error) {
	as, err := p.assetProcessor.GetByCompartmentId(m.Id())
	if err != nil {
		return Model{}, err
	}
	return Clone(m).SetAssets(as).Build(), nil
}

func (p *Processor) Create(mb *message.Buffer) func(characterId uint32, inventoryType inventory.Type, capacity uint32) (Model, error) {
	return func(characterId uint32, inventoryType inventory.Type, capacity uint32) (Model, error) {
		p.l.Debugf("Attempting to create compartment of type [%d] for character [%d] with capacity [%d].", inventoryType, characterId, capacity)
		var c Model
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			var err error
			c, err = create(tx, p.t.Id(), characterId, inventoryType, capacity)
			if err != nil {
				return err
			}
			return mb.Put(compartment.EnvEventTopicStatus, CreatedEventStatusProvider(c.Id(), characterId, c.Type(), c.Capacity()))
		})
		if txErr != nil {
			return Model{}, txErr
		}
		p.l.Debugf("Created compartment [%s] for character [%d] with capacity [%d].", c.Id().String(), characterId, capacity)
		return c, nil
	}
}

func (p *Processor) DeleteByModel(mb *message.Buffer) func(c Model) error {
	return func(c Model) error {
		p.l.Debugf("Attempting to delete compartment [%s].", c.Id().String())
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			err := model.ForEachSlice(model.FixedProvider(c.Assets()), p.assetProcessor.WithTransaction(tx).Delete(mb)(c.CharacterId(), c.Id()))
			if err != nil {
				return err
			}
			err = deleteById(tx, p.t.Id(), c.Id())
			if err != nil {
				return err
			}
			return mb.Put(compartment.EnvEventTopicStatus, DeletedEventStatusProvider(c.Id(), c.CharacterId()))
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to delete compartment [%s].", c.Id().String())
			return txErr
		}
		p.l.Debugf("Deleted compartment [%s].", c.Id().String())
		return nil
	}
}

func temporarySlot() int16 {
	return int16(math.MinInt16)
}

func (p *Processor) EquipItemAndEmit(characterId uint32, source int16, destination int16) error {
	return message.Emit(p.producer)(model.Flip(model.Flip(model.Flip(p.EquipItem)(characterId))(source))(destination))
}

func (p *Processor) EquipItem(mb *message.Buffer) func(characterId uint32) func(source int16) func(destination int16) error {
	return func(characterId uint32) func(source int16) func(destination int16) error {
		return func(source int16) func(destination int16) error {
			return func(destination int16) error {
				p.l.Debugf("Attempting to equip item in slot [%d] to [%d] for character [%d].", source, destination, characterId)
				invLock := LockRegistry().Get(characterId, inventory.TypeValueEquip)
				invLock.Lock()
				defer invLock.Unlock()

				var a1 asset.Model[any]
				var actualDestination int16
				txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
					var c Model
					var err error
					c, err = p.WithTransaction(tx).GetByCharacterAndType(characterId)(inventory.TypeValueEquip)
					if err != nil {
						p.l.WithError(err).Errorf("Unable to get compartment by type [%d] for character [%d].", inventory.TypeValueEquip, characterId)
						return err
					}

					assetProvider := p.assetProcessor.WithTransaction(tx).BySlotProvider(c.Id())
					a1, err = assetProvider(source)()
					if err != nil {
						p.l.WithError(err).Errorf("Unable to get asset in compartment [%d] by slot [%d].", c.Id(), source)
						return err
					}
					p.l.Debugf("Character [%d] is attempting to equip item [%d].", characterId, a1.TemplateId())
					actualDestination, err = p.equipmentProcessor.DestinationSlotProvider(destination)(a1.TemplateId())()
					if err != nil {
						p.l.WithError(err).Errorf("Unable to determine actual destination for item being equipped.")
						return err
					}
					p.l.Debugf("Character [%d] moving asset from [%d] to [%d] if present.", characterId, actualDestination, temporarySlot())
					err = p.assetProcessor.WithTransaction(tx).UpdateSlot(mb)(characterId, c.Id(), assetProvider(actualDestination), model.FixedProvider(temporarySlot()))
					if err != nil {
						p.l.WithError(err).Errorf("Unable to update asset slot from [%d] to [%d]. Character [%d]. Compartment [%d].", actualDestination, temporarySlot(), characterId, c.Id())
						return err
					}
					p.l.Debugf("Character [%d] moving asset from source [%d] to destination [%d].", characterId, a1.Slot(), actualDestination)
					err = p.assetProcessor.WithTransaction(tx).UpdateSlot(mb)(characterId, c.Id(), model.FixedProvider(a1), model.FixedProvider(actualDestination))
					if err != nil {
						p.l.WithError(err).Errorf("Unable to update asset slot from [%d] to [%d]. Character [%d]. Compartment [%d].", a1.Slot(), actualDestination, characterId, c.Id())
						return err
					}
					p.l.Debugf("Character [%d] moving asset from [%d] to [%d] if present.", characterId, temporarySlot(), source)
					err = p.assetProcessor.WithTransaction(tx).UpdateSlot(mb)(characterId, c.Id(), assetProvider(temporarySlot()), model.FixedProvider(source))
					if err != nil {
						p.l.WithError(err).Errorf("Unable to update asset slot from [%d] to [%d]. Character [%d]. Compartment [%d].", temporarySlot(), source, characterId, c.Id())
						return err
					}

					p.l.Debugf("Now verifying other inventory operations that may be necessary.")

					if item.GetClassification(item.Id(a1.TemplateId())) == item.ClassificationOverall {
						var ps slot.Slot
						ps, err = slot.GetSlotByType("pants")
						if err != nil {
							p.l.WithError(err).Errorf("Unable to get slot by type [pants].")
							return err
						}
						var nfs int16
						nfs, err = c.NextFreeSlot()
						if err != nil {
							p.l.WithError(err).Errorf("No free slots for pants.")
							return err
						}
						err = p.assetProcessor.WithTransaction(tx).UpdateSlot(mb)(characterId, c.Id(), assetProvider(int16(ps.Position)), model.FixedProvider(nfs))
						if err != nil {
							p.l.WithError(err).Errorf("Unable to update asset slot from [%d] to [%d]. Character [%d]. Compartment [%d].", ps.Position, nfs, characterId, c.Id())
							return err
						}
					}

					if item.GetClassification(item.Id(a1.TemplateId())) == item.Classification(106) {
						var ts slot.Slot
						ts, err = slot.GetSlotByType("top")
						if err != nil {
							p.l.WithError(err).Errorf("Unable to get slot by type [top].")
							return err
						}
						var ta asset.Model[any]
						ta, err = assetProvider(int16(ts.Position))()
						if err == nil {
							if item.GetClassification(item.Id(ta.TemplateId())) == item.ClassificationOverall {
								var nfs int16
								nfs, err = c.NextFreeSlot()
								if err != nil {
									p.l.WithError(err).Errorf("No free slots for top.")
									return err
								}
								err = p.assetProcessor.WithTransaction(tx).UpdateSlot(mb)(characterId, c.Id(), model.FixedProvider(ta), model.FixedProvider(nfs))
								if err != nil {
									p.l.WithError(err).Errorf("Unable to update asset slot from [%d] to [%d]. Character [%d]. Compartment [%d].", ts.Position, nfs, characterId, c.Id())
									return err
								}
							}
						}
					}
					return nil
				})
				if txErr != nil {
					p.l.Debugf("Unable to equip item in slot [%d] to [%d] for character [%d].", source, actualDestination, characterId)
				}
				p.l.Debugf("Character [%d] equipped item [%d] in slot [%d].", characterId, a1.TemplateId(), actualDestination)
				return nil
			}
		}
	}
}

func (p *Processor) RemoveEquipAndEmit(characterId uint32, source int16, destination int16) error {
	return message.Emit(p.producer)(model.Flip(model.Flip(model.Flip(p.RemoveEquip)(characterId))(source))(destination))
}

func (p *Processor) RemoveEquip(mb *message.Buffer) func(characterId uint32) func(source int16) func(destination int16) error {
	return func(characterId uint32) func(source int16) func(destination int16) error {
		return func(source int16) func(destination int16) error {
			return func(destination int16) error {
				p.l.Debugf("Attempting to remove equipment in slot [%d] to [%d] for character [%d].", source, destination, characterId)
				invLock := LockRegistry().Get(characterId, inventory.TypeValueEquip)
				invLock.Lock()
				defer invLock.Unlock()

				txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
					var c Model
					var err error
					c, err = p.WithTransaction(tx).GetByCharacterAndType(characterId)(inventory.TypeValueEquip)
					if err != nil {
						p.l.WithError(err).Errorf("Unable to get compartment by type [%d] for character [%d].", inventory.TypeValueEquip, characterId)
						return err
					}

					var fsp model.Provider[int16]
					assetProvider := p.assetProcessor.WithTransaction(tx).BySlotProvider(c.Id())
					if destination > 0 && uint32(destination) < c.Capacity() {
						_, err = assetProvider(destination)()
						if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
							fsp = model.FixedProvider(destination)
						}
					}
					if fsp == nil {
						p.l.Debugf("Desired free slot [%d] is occupied. Checking next free slot.", destination)
						var nfs int16
						nfs, err = c.NextFreeSlot()
						if err != nil {
							p.l.WithError(err).Errorf("No free slots exist for equip. Cannot remove equipment.")
							return err
						}
						fsp = model.FixedProvider(nfs)
					}
					err = p.assetProcessor.WithTransaction(tx).UpdateSlot(mb)(characterId, c.Id(), assetProvider(source), fsp)
					if err != nil {
						ds, _ := fsp()
						p.l.WithError(err).Errorf("Unable to update asset slot from [%d] to [%d]. Character [%d]. Compartment [%d].", source, ds, characterId, c.Id())
						return err
					}
					return nil
				})
				if txErr != nil {
					p.l.WithError(txErr).Errorf("Unable to remove equipment in slot [%d] for character [%d].", source, characterId)
					return txErr
				}
				return nil
			}
		}
	}
}

func (p *Processor) MoveAndEmit(characterId uint32, inventoryType inventory.Type, source int16, destination int16) error {
	return message.Emit(p.producer)(model.Flip(model.Flip(model.Flip(model.Flip(p.MoveAndLock)(characterId))(inventoryType))(source))(destination))
}

func (p *Processor) MoveAndLock(mb *message.Buffer) func(characterId uint32) func(inventoryType inventory.Type) func(source int16) func(destination int16) error {
	return func(characterId uint32) func(inventoryType inventory.Type) func(source int16) func(destination int16) error {
		return func(inventoryType inventory.Type) func(source int16) func(destination int16) error {
			return func(source int16) func(destination int16) error {
				return func(destination int16) error {
					invLock := LockRegistry().Get(characterId, inventoryType)
					invLock.Lock()
					defer invLock.Unlock()
					return p.Move(mb)(characterId)(inventoryType)(source)(destination)
				}
			}
		}
	}
}

func (p *Processor) Move(mb *message.Buffer) func(characterId uint32) func(inventoryType inventory.Type) func(source int16) func(destination int16) error {
	return func(characterId uint32) func(inventoryType inventory.Type) func(source int16) func(destination int16) error {
		return func(inventoryType inventory.Type) func(source int16) func(destination int16) error {
			return func(source int16) func(destination int16) error {
				return func(destination int16) error {
					p.l.Debugf("Attempting to move asset in slot [%d] to [%d] for character [%d].", source, destination, characterId)

					var a1 asset.Model[any]
					txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
						// Get compartment
						c, err := p.WithTransaction(tx).GetByCharacterAndType(characterId)(inventoryType)
						if err != nil {
							p.l.WithError(err).Errorf("Unable to get compartment by type [%d] for character [%d].", inventoryType, characterId)
							return err
						}

						// Get source asset
						assetProvider := p.assetProcessor.WithTransaction(tx).BySlotProvider(c.Id())
						a1, err = assetProvider(source)()
						if err != nil {
							p.l.WithError(err).Errorf("Unable to get asset in compartment [%d] by slot [%d].", c.Id(), source)
							return err
						}
						p.l.Debugf("Character [%d] is attempting to move asset [%d].", characterId, a1.TemplateId())

						// Check if there's an asset at the destination slot
						a2, err := assetProvider(destination)()
						if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
							p.l.WithError(err).Errorf("Error checking asset in compartment [%d] by slot [%d].", c.Id(), destination)
							return err
						}

						// Determine if we should merge or swap
						if err == nil && p.canMergeAssets(inventoryType, a1, a2, characterId) {
							return p.WithTransaction(tx).mergeAssets(mb)(characterId, c, a1, a2, source, destination)
						}

						// Default to swap logic
						return p.WithTransaction(tx).swapAssets(mb)(characterId, c, assetProvider, a1, source, destination)
					})

					if txErr != nil {
						p.l.Debugf("Unable to move asset in slot [%d] to [%d] for character [%d].", source, destination, characterId)
						return txErr
					}

					p.l.Debugf("Character [%d] moved asset [%d] to slot [%d].", characterId, a1.TemplateId(), destination)
					return nil
				}
			}
		}
	}
}

// swapAssets handles swapping two assets between slots
func (p *Processor) swapAssets(mb *message.Buffer) func(characterId uint32, c Model, assetProvider func(int16) model.Provider[asset.Model[any]], a1 asset.Model[any], source int16, destination int16) error {
	return func(characterId uint32, c Model, assetProvider func(int16) model.Provider[asset.Model[any]], a1 asset.Model[any], source int16, destination int16) error {
		// Move destination asset to temporary slot
		err := p.assetProcessor.WithTransaction(p.db).UpdateSlot(mb)(characterId, c.Id(), assetProvider(destination), model.FixedProvider(temporarySlot()))
		if err != nil {
			p.l.WithError(err).Errorf("Unable to update asset slot from [%d] to [%d]. Character [%d]. Compartment [%d].", destination, temporarySlot(), characterId, c.Id())
			return err
		}

		// Move source asset to destination
		err = p.assetProcessor.WithTransaction(p.db).UpdateSlot(mb)(characterId, c.Id(), model.FixedProvider(a1), model.FixedProvider(destination))
		if err != nil {
			p.l.WithError(err).Errorf("Unable to update asset slot from [%d] to [%d]. Character [%d]. Compartment [%d].", a1.Slot(), destination, characterId, c.Id())
			return err
		}

		// Move temporary asset to source
		err = p.assetProcessor.WithTransaction(p.db).UpdateSlot(mb)(characterId, c.Id(), assetProvider(temporarySlot()), model.FixedProvider(source))
		if err != nil {
			p.l.WithError(err).Errorf("Unable to update asset slot from [%d] to [%d]. Character [%d]. Compartment [%d].", temporarySlot(), source, characterId, c.Id())
			return err
		}

		GetReservationRegistry().SwapReservation(p.t, characterId, c.Type(), source, destination)
		return nil
	}
}

// mergeAssets handles merging two assets with the same template ID
func (p *Processor) mergeAssets(mb *message.Buffer) func(characterId uint32, c Model, a1 asset.Model[any], a2 asset.Model[any], source int16, destination int16) error {
	return func(characterId uint32, c Model, a1 asset.Model[any], a2 asset.Model[any], source int16, destination int16) error {

		// Get slot max for the item
		slotMax, err := p.assetProcessor.GetSlotMax(a1.TemplateId())
		if err != nil {
			p.l.WithError(err).Errorf("Unable to get slot max for item [%d].", a1.TemplateId())
			return err
		}

		totalQuantity := a1.Quantity() + a2.Quantity()

		// If the total quantity fits in one slot
		if totalQuantity <= slotMax {
			// Update destination quantity
			err = p.assetProcessor.WithTransaction(p.db).UpdateQuantity(mb)(characterId, c.Id(), a2, totalQuantity)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to update quantity of asset [%d] to [%d].", a2.Id(), totalQuantity)
				return err
			}

			// Delete source asset
			err = p.assetProcessor.WithTransaction(p.db).Delete(mb)(characterId, c.Id())(a1)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to delete asset [%d].", a1.Id())
				return err
			}

			p.l.Debugf("Character [%d] merged asset [%d] into asset [%d] with total quantity [%d].",
				characterId, a1.Id(), a2.Id(), totalQuantity)
			return nil
		}

		// Fill destination to max and keep remainder in source
		err = p.assetProcessor.WithTransaction(p.db).UpdateQuantity(mb)(characterId, c.Id(), a2, slotMax)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to update quantity of asset [%d] to [%d].", a2.Id(), slotMax)
			return err
		}

		// Update source with remaining quantity
		remainingQuantity := totalQuantity - slotMax
		err = p.assetProcessor.WithTransaction(p.db).UpdateQuantity(mb)(characterId, c.Id(), a1, remainingQuantity)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to update quantity of asset [%d] to [%d].", a1.Id(), remainingQuantity)
			return err
		}

		p.l.Debugf("Character [%d] filled asset [%d] to max [%d] and kept [%d] in asset [%d].",
			characterId, a2.Id(), slotMax, remainingQuantity, a1.Id())
		return nil
	}
}

// canMergeAssets checks if two assets can be merged based on the specified rules
func (p *Processor) canMergeAssets(inventoryType inventory.Type, sourceAsset asset.Model[any], destAsset asset.Model[any], characterId uint32) bool {
	// Rule 1: Inventories of type Equip cannot support merging
	if inventoryType == inventory.TypeValueEquip {
		return false
	}

	// Rule 2: Assets must have the same template ID
	if sourceAsset.TemplateId() != destAsset.TemplateId() {
		return false
	}

	// Rule 3: In inventories of type Use, rechargeable assets cannot be stacked
	if inventoryType == inventory.TypeValueUse {
		// Check if either asset is rechargeable
		if sourceAsset.IsConsumable() {
			sourceRefData, ok := sourceAsset.ReferenceData().(asset.ConsumableReferenceData)
			if ok && sourceRefData.Rechargeable() > 0 {
				return false
			}
		}
		if destAsset.IsConsumable() {
			destRefData, ok := destAsset.ReferenceData().(asset.ConsumableReferenceData)
			if ok && destRefData.Rechargeable() > 0 {
				return false
			}
		}
	}

	// Rule 4: Neither asset can have an active reservation
	sourceReserved := GetReservationRegistry().GetReservedQuantity(p.t, characterId, inventoryType, sourceAsset.Slot())
	destReserved := GetReservationRegistry().GetReservedQuantity(p.t, characterId, inventoryType, destAsset.Slot())
	if sourceReserved > 0 || destReserved > 0 {
		return false
	}

	// Rule 5: Check if both assets have quantity (are stackable)
	if !sourceAsset.HasQuantity() || !destAsset.HasQuantity() {
		return false
	}

	// TODO: Rule 6: Assets must have the same owner to be stackable

	// Rule 7: Check if destination asset has already reached its slot max
	slotMax, err := p.assetProcessor.GetSlotMax(destAsset.TemplateId())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get slot max for item [%d].", destAsset.TemplateId())
		return false
	}

	if destAsset.Quantity() >= slotMax {
		return false
	}

	return true
}

func (p *Processor) IncreaseCapacityAndEmit(characterId uint32, inventoryType inventory.Type, amount uint32) error {
	return message.Emit(p.producer)(model.Flip(model.Flip(model.Flip(p.IncreaseCapacity)(characterId))(inventoryType))(amount))
}

func (p *Processor) IncreaseCapacity(mb *message.Buffer) func(characterId uint32) func(inventoryType inventory.Type) func(amount uint32) error {
	return func(characterId uint32) func(inventoryType inventory.Type) func(amount uint32) error {
		return func(inventoryType inventory.Type) func(amount uint32) error {
			return func(amount uint32) error {
				p.l.Debugf("Character [%d] attempting to change compartment capacity by [%d]. Type [%d].", characterId, amount, inventoryType)
				invLock := LockRegistry().Get(characterId, inventoryType)
				invLock.Lock()
				defer invLock.Unlock()

				var capacity uint32
				txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
					c, err := p.GetByCharacterAndType(characterId)(inventoryType)
					if err != nil {
						return err
					}
					capacity = uint32(math.Min(96, float64(c.Capacity()+amount)))
					_, err = updateCapacity(tx, p.t.Id(), characterId, int8(inventoryType), capacity)
					if err != nil {
						return err
					}
					return mb.Put(compartment.EnvEventTopicStatus, CapacityChangedEventStatusProvider(c.Id(), characterId, inventoryType, capacity))
				})
				if txErr != nil {
					p.l.WithError(txErr).Errorf("Character [%d] unable to change compartment capacity. Type [%d].", characterId, inventoryType)
					return txErr
				}
				p.l.Debugf("Character [%d] changed compartment capacity by [%d]. Type [%d].", characterId, amount, inventoryType)
				return nil
			}
		}
	}
}

func (p *Processor) DropAndEmit(characterId uint32, inventoryType inventory.Type, m _map.Model, x int16, y int16, source int16, quantity int16) error {
	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return p.Drop(buf)(characterId, inventoryType, m, x, y, source, quantity)
	})
}

func (p *Processor) Drop(mb *message.Buffer) func(characterId uint32, inventoryType inventory.Type, m _map.Model, x int16, y int16, source int16, quantity int16) error {
	return func(characterId uint32, inventoryType inventory.Type, m _map.Model, x int16, y int16, source int16, quantity int16) error {
		p.l.Debugf("Character [%d] attempting to drop [%d] asset from slot [%d].", characterId, quantity, source)
		if quantity < 0 {
			return errors.New("cannot drop negative quantity")
		}
		if quantity == 0 {
			return errors.New("cannot drop nothing")
		}

		invLock := LockRegistry().Get(characterId, inventoryType)
		invLock.Lock()
		defer invLock.Unlock()

		var a asset.Model[any]
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			c, err := p.GetByCharacterAndType(characterId)(inventoryType)
			if err != nil {
				return err
			}
			a, err = p.assetProcessor.WithTransaction(tx).GetBySlot(c.Id(), source)
			if err != nil {
				return err
			}
			reservedQty := GetReservationRegistry().GetReservedQuantity(p.t, characterId, inventoryType, source)
			initialQty := a.Quantity() - reservedQty

			if initialQty < uint32(quantity) {
				return errors.New("cannot drop more than what is owned")
			}
			if initialQty == uint32(quantity) {
				err = p.assetProcessor.WithTransaction(tx).Drop(mb)(characterId, c.Id())(a)
				if err != nil {
					return err
				}
				return nil
			}
			newQuantity := a.Quantity() - uint32(quantity)
			err = p.assetProcessor.WithTransaction(tx).UpdateQuantity(mb)(characterId, c.Id(), a, newQuantity)
			if err != nil {
				return err
			}
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Character [%d] unable to drop [%d] asset from slot [%d].", characterId, quantity, source)
			return txErr
		}
		p.l.Debugf("Character [%d] dropped [%d] asset [%d] from slot [%d].", characterId, quantity, a.Id(), source)
		if inventoryType == inventory.TypeValueEquip {
			return p.dropProcessor.CreateForEquipment(mb)(m, a.TemplateId(), a.ReferenceId(), 2, x, y, characterId)
		} else {
			return p.dropProcessor.CreateForItem(mb)(m, a.TemplateId(), uint32(math.Abs(float64(quantity))), 2, x, y, characterId)
		}
	}
}

func (p *Processor) RequestReserveAndEmit(characterId uint32, inventoryType inventory.Type, reservationRequests []ReservationRequest, transactionId uuid.UUID) error {
	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return p.RequestReserve(buf)(characterId, inventoryType, reservationRequests, transactionId)
	})
}

func (p *Processor) RequestReserve(mb *message.Buffer) func(characterId uint32, inventoryType inventory.Type, reservationRequests []ReservationRequest, transactionId uuid.UUID) error {
	return func(characterId uint32, inventoryType inventory.Type, reservationRequests []ReservationRequest, transactionId uuid.UUID) error {
		p.l.Debugf("Character [%d] attempting to reserve [%d] inventory [%d] reservation [%s].", characterId, len(reservationRequests), inventoryType, transactionId.String())
		invLock := LockRegistry().Get(characterId, inventoryType)
		invLock.Lock()
		defer invLock.Unlock()

		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetByCharacterAndType(characterId)(inventoryType)
			if err != nil {
				return err
			}
			for _, request := range reservationRequests {
				var a asset.Model[any]
				a, err = p.assetProcessor.WithTransaction(tx).GetBySlot(c.Id(), request.Slot)
				if err != nil {
					return err
				}
				if a.TemplateId() != request.ItemId {
					return errors.New("cannot reserve non-existent item")
				}
				currentReservedQty := GetReservationRegistry().GetReservedQuantity(p.t, characterId, inventoryType, request.Slot)
				if a.Quantity()-currentReservedQty < uint32(request.Quantity) {
					return errors.New("cannot reserve more than what is owned")
				}
				_, err = GetReservationRegistry().AddReservation(p.t, transactionId, characterId, inventoryType, request.Slot, request.ItemId, uint32(request.Quantity), time.Second*time.Duration(30))
				if err != nil {
					return err
				}
				return mb.Put(compartment.EnvEventTopicStatus, ReservedEventStatusProvider(c.Id(), characterId, request.ItemId, request.Slot, uint32(request.Quantity), transactionId))
			}
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Character [%d] unable to reserve [%d] inventory [%d] via reservation request [%s].", characterId, len(reservationRequests), inventoryType, transactionId.String())
			return txErr
		}
		return nil
	}
}

func (p *Processor) CancelReservationAndEmit(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return p.CancelReservation(buf)(characterId, inventoryType, transactionId, slot)
	})
}

func (p *Processor) CancelReservation(mb *message.Buffer) func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
	return func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
		p.l.Debugf("Character [%d] attempting to cancel inventory [%d] reservation [%s].", characterId, inventoryType, transactionId.String())
		invLock := LockRegistry().Get(characterId, inventoryType)
		invLock.Lock()
		defer invLock.Unlock()

		c, err := p.GetByCharacterAndType(characterId)(inventoryType)
		if err != nil {
			return err
		}

		res, err := GetReservationRegistry().RemoveReservation(p.t, transactionId, characterId, inventoryType, slot)
		if err != nil {
			return nil
		}
		return mb.Put(compartment.EnvEventTopicStatus, ReservationCancelledEventStatusProvider(c.Id(), characterId, res.ItemId(), slot, res.Quantity()))
	}
}

func (p *Processor) ConsumeAssetAndEmit(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return p.ConsumeAsset(buf)(characterId, inventoryType, transactionId, slot)
	})
}

func (p *Processor) ConsumeAsset(mb *message.Buffer) func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
	return func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
		p.l.Debugf("Character [%d] attempting to consume asset in inventory [%d] slot [%d]. Transaction [%s].", characterId, inventoryType, slot, transactionId.String())
		invLock := LockRegistry().Get(characterId, inventoryType)
		invLock.Lock()
		defer invLock.Unlock()

		res, err := GetReservationRegistry().RemoveReservation(p.t, transactionId, characterId, inventoryType, slot)
		if err != nil {
			return nil
		}

		var a asset.Model[any]
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			var c Model
			c, err = p.WithTransaction(tx).GetByCharacterAndType(characterId)(inventoryType)
			if err != nil {
				return err
			}
			a, err = p.assetProcessor.WithTransaction(tx).GetBySlot(c.Id(), slot)
			if err != nil {
				return err
			}
			reservedQty := GetReservationRegistry().GetReservedQuantity(p.t, characterId, inventoryType, slot)
			initialQty := a.Quantity() - reservedQty
			if initialQty <= 1 {
				err = p.assetProcessor.WithTransaction(tx).Delete(mb)(characterId, c.Id())(a)
				if err != nil {
					return err
				}
				return nil
			}
			newQuantity := a.Quantity() - res.Quantity()
			err = p.assetProcessor.WithTransaction(tx).UpdateQuantity(mb)(characterId, c.Id(), a, newQuantity)
			if err != nil {
				return err
			}
			return nil
		})
		if txErr != nil {
			p.l.WithError(err).Errorf("Character [%d] unable to consume asset in inventory [%d] slot [%d]. Transaction [%d].", characterId, inventoryType, transactionId, slot)
			return txErr
		}
		p.l.Debugf("Character [%d] consumed [%d] of item [%d].", characterId, res.Quantity(), a.TemplateId())
		return nil
	}
}

func (p *Processor) DestroyAssetAndEmit(characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error {
	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return p.DestroyAsset(buf)(characterId, inventoryType, slot, quantity)
	})
}

func (p *Processor) DestroyAsset(mb *message.Buffer) func(characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error {
	return func(characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error {
		p.l.Debugf("Character [%d] attempting to destroy [%d] asset in inventory [%d] slot [%d].", characterId, quantity, inventoryType, slot)
		invLock := LockRegistry().Get(characterId, inventoryType)
		invLock.Lock()
		defer invLock.Unlock()

		var a asset.Model[any]
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetByCharacterAndType(characterId)(inventoryType)
			if err != nil {
				return err
			}
			a, err = p.assetProcessor.WithTransaction(tx).GetBySlot(c.Id(), slot)
			if err != nil {
				return err
			}

			// If the asset doesn't have quantity, or if the asset's quantity is less than or equal to the quantity provided, delete it
			if !a.HasQuantity() || a.Quantity() <= quantity {
				err = p.assetProcessor.WithTransaction(tx).Delete(mb)(characterId, c.Id())(a)
				if err != nil {
					return err
				}
			} else {
				// If the asset has quantity, and it's greater than the quantity provided, update the quantity
				newQuantity := a.Quantity() - quantity
				err = p.assetProcessor.WithTransaction(tx).UpdateQuantity(mb)(characterId, c.Id(), a, newQuantity)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Character [%d] unable to destroy asset in inventory [%d] slot [%d].", characterId, inventoryType, slot)
			return txErr
		}
		p.l.Debugf("Character [%d] destroyed asset [%d].", characterId, a.Id())
		return nil
	}
}

func (p *Processor) CreateAssetAndEmit(characterId uint32, inventoryType inventory.Type, templateId uint32, quantity uint32, expiration time.Time, ownerId uint32, flag uint16, rechargeable uint64) error {
	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return p.CreateAsset(buf)(characterId, inventoryType, templateId, quantity, expiration, ownerId, flag, rechargeable)
	})
}

func (p *Processor) CreateAsset(mb *message.Buffer) func(characterId uint32, inventoryType inventory.Type, templateId uint32, quantity uint32, expiration time.Time, ownerId uint32, flag uint16, rechargeable uint64) error {
	return func(characterId uint32, inventoryType inventory.Type, templateId uint32, quantity uint32, expiration time.Time, ownerId uint32, flag uint16, rechargeable uint64) error {
		p.l.Debugf("Character [%d] attempting to create asset in inventory [%d].", characterId, inventoryType)
		invLock := LockRegistry().Get(characterId, inventoryType)
		invLock.Lock()
		defer invLock.Unlock()

		var a asset.Model[any]
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetByCharacterAndType(characterId)(inventoryType)
			if err != nil {
				return err
			}
			nfs, err := c.NextFreeSlot()
			if err != nil {
				return err
			}
			a, err = p.assetProcessor.WithTransaction(tx).Create(mb)(characterId, c.Id(), templateId, nfs, quantity, expiration, ownerId, flag, rechargeable)
			if err != nil {
				return err
			}
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Character [%d] unable to create asset in inventory [%d].", characterId, inventoryType)
			return txErr
		}
		p.l.Debugf("Character [%d] created asset [%d].", characterId, a.Id())
		return nil
	}
}

func (p *Processor) AttemptEquipmentPickUpAndEmit(m _map.Model, characterId uint32, dropId uint32, templateId uint32, referenceId uint32) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.AttemptEquipmentPickUp(buf)(m, characterId, dropId, templateId, referenceId)
	})
}

func (p *Processor) AttemptEquipmentPickUp(mb *message.Buffer) func(m _map.Model, characterId uint32, dropId uint32, templateId uint32, referenceId uint32) error {
	return func(m _map.Model, characterId uint32, dropId uint32, templateId uint32, referenceId uint32) error {

		inventoryType, ok := inventory.TypeFromItemId(item.Id(templateId))
		if !ok {
			return errors.New("invalid inventory item")
		}

		if inventoryType != inventory.TypeValueEquip {
			p.l.Errorf("Provided inventoryType [%d] does not match expected one [%d] for itemId [%d].", inventoryType, 1, templateId)
			return errors.New("invalid inventory type")
		}

		p.l.Debugf("Gaining [%d] item [%d] for character [%d] in inventory [%d].", 1, templateId, characterId, inventoryType)
		invLock := LockRegistry().Get(characterId, inventoryType)
		invLock.Lock()
		defer invLock.Unlock()

		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			c, err := p.GetByCharacterAndType(characterId)(inventoryType)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to locate inventory [%d] for character [%d].", inventoryType, characterId)
				return err
			}

			s, err := c.NextFreeSlot()
			if err != nil {
				return err
			}

			_, err = p.assetProcessor.WithTransaction(tx).Acquire(mb)(characterId, c.Id(), templateId, s, 1, referenceId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to create [%d] equipable [%d] for character [%d].", 1, templateId, characterId)
				return err
			}
			return nil
		})
		if txErr != nil {
			mb = message.NewBuffer()
			return p.dropProcessor.CancelReservation(mb)(m, dropId, characterId)
		}
		return p.dropProcessor.RequestPickUp(mb)(m, dropId, characterId)
	}
}

func (p *Processor) AttemptItemPickUpAndEmit(m _map.Model, characterId uint32, dropId uint32, templateId uint32, quantity uint32) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		return p.AttemptItemPickUp(buf)(m, characterId, dropId, templateId, quantity)
	})
}

func (p *Processor) AttemptItemPickUp(mb *message.Buffer) func(m _map.Model, characterId uint32, dropId uint32, templateId uint32, quantity uint32) error {
	return func(m _map.Model, characterId uint32, dropId uint32, templateId uint32, quantity uint32) error {
		inventoryType, ok := inventory.TypeFromItemId(item.Id(templateId))
		if !ok {
			return errors.New("invalid inventory item")
		}

		invLock := LockRegistry().Get(characterId, inventoryType)
		invLock.Lock()
		defer invLock.Unlock()

		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			// Get the compartment for the character and inventory type
			c, err := p.WithTransaction(tx).GetByCharacterAndType(characterId)(inventoryType)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to get compartment by type [%d] for character [%d].", inventoryType, characterId)
				return err
			}

			// Get all assets in the compartment
			assets, err := p.assetProcessor.WithTransaction(tx).GetByCompartmentId(c.Id())
			if err != nil {
				p.l.WithError(err).Errorf("Unable to get assets in compartment [%s].", c.Id())
				return err
			}

			// Get the slot max for this item
			slotMax, err := p.assetProcessor.GetSlotMax(templateId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to get slot max for item [%d].", templateId)
				return err
			}

			// Check if any existing asset has the same templateId and can be stacked
			var assetToUpdate asset.Model[any]
			for _, a := range assets {
				if a.TemplateId() == templateId && a.HasQuantity() && a.Quantity() < slotMax {
					assetToUpdate = a
					break
				}
			}

			if assetToUpdate.Id() != 0 {
				// Calculate new quantity
				newQuantity := assetToUpdate.Quantity() + quantity

				// Check if the new quantity exceeds the slot max
				if newQuantity > slotMax {
					// Split the quantity
					remainingQuantity := newQuantity - slotMax

					// Update the existing asset to max
					err = p.assetProcessor.WithTransaction(tx).UpdateQuantity(mb)(characterId, c.Id(), assetToUpdate, slotMax)
					if err != nil {
						p.l.WithError(err).Errorf("Unable to update quantity of asset [%d] to [%d].", assetToUpdate.Id(), slotMax)
						return err
					}
					p.l.Debugf("Character [%d] increased quantity of asset [%d] to max [%d].", characterId, assetToUpdate.Id(), slotMax)

					// Create a new asset with the remaining quantity
					err = p.CreateAsset(mb)(characterId, inventoryType, templateId, remainingQuantity, time.Time{}, 0, 0, 0)
					if err != nil {
						p.l.WithError(err).Errorf("Unable to create asset [%d] for character [%d] with remaining quantity [%d].", templateId, characterId, remainingQuantity)
						return err
					}
				} else {
					// Update the quantity of the existing asset
					err = p.assetProcessor.WithTransaction(tx).UpdateQuantity(mb)(characterId, c.Id(), assetToUpdate, newQuantity)
					if err != nil {
						p.l.WithError(err).Errorf("Unable to update quantity of asset [%d] to [%d].", assetToUpdate.Id(), newQuantity)
						return err
					}
					p.l.Debugf("Character [%d] increased quantity of asset [%d] to [%d].", characterId, assetToUpdate.Id(), newQuantity)
				}
			} else {
				// Create a new asset
				err = p.CreateAsset(mb)(characterId, inventoryType, templateId, quantity, time.Time{}, 0, 0, 0)
				if err != nil {
					p.l.WithError(err).Errorf("Unable to create asset [%d] for character [%d].", templateId, characterId)
					return err
				}
			}
			return nil
		})

		if txErr != nil {
			mb = message.NewBuffer()
			return p.dropProcessor.CancelReservation(mb)(m, dropId, characterId)
		}
		return p.dropProcessor.RequestPickUp(mb)(m, dropId, characterId)
	}
}

func (p *Processor) RechargeAssetAndEmit(characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error {
	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return p.RechargeAsset(buf)(characterId, inventoryType, slot, quantity)
	})
}

func (p *Processor) RechargeAsset(mb *message.Buffer) func(characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error {
	return func(characterId uint32, inventoryType inventory.Type, slot int16, quantity uint32) error {
		p.l.Debugf("Character [%d] attempting to recharge asset in inventory [%d] slot [%d] with quantity [%d].", characterId, inventoryType, slot, quantity)

		// Only TypeValueUse compartment type should support this functionality
		if inventoryType != inventory.TypeValueUse {
			p.l.Errorf("Recharge operation not supported for inventory type [%d]. Only TypeValueUse is supported.", inventoryType)
			return errors.New("recharge operation not supported for this inventory type")
		}

		invLock := LockRegistry().Get(characterId, inventoryType)
		invLock.Lock()
		defer invLock.Unlock()

		var a asset.Model[any]
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetByCharacterAndType(characterId)(inventoryType)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to get compartment by type [%d] for character [%d].", inventoryType, characterId)
				return err
			}

			// Ensure the item exists in the compartment
			a, err = p.assetProcessor.WithTransaction(tx).GetBySlot(c.Id(), slot)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to get asset in compartment [%s] by slot [%d].", c.Id(), slot)
				return err
			}

			// Update the quantity with the provided quantity
			newQuantity := a.Quantity() + quantity
			err = p.assetProcessor.WithTransaction(tx).UpdateQuantity(mb)(characterId, c.Id(), a, newQuantity)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to update quantity of asset [%d] to [%d].", a.Id(), newQuantity)
				return err
			}

			return nil
		})

		if txErr != nil {
			p.l.WithError(txErr).Errorf("Character [%d] unable to recharge asset in inventory [%d] slot [%d].", characterId, inventoryType, slot)
			return txErr
		}

		p.l.Debugf("Character [%d] recharged asset [%d] with quantity [%d].", characterId, a.Id(), quantity)
		return nil
	}
}

func (p *Processor) MergeAndCompactAndEmit(characterId uint32, inventoryType inventory.Type) error {
	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return p.MergeAndCompact(buf)(characterId, inventoryType)
	})
}

func (p *Processor) CompactAndSortAndEmit(characterId uint32, inventoryType inventory.Type) error {
	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return p.CompactAndSort(buf)(characterId, inventoryType)
	})
}

func (p *Processor) MergeAndCompact(mb *message.Buffer) func(characterId uint32, inventoryType inventory.Type) error {
	return func(characterId uint32, inventoryType inventory.Type) error {
		p.l.Debugf("Character [%d] attempting to merge and compact assets in inventory [%d].", characterId, inventoryType)

		invLock := LockRegistry().Get(characterId, inventoryType)
		invLock.Lock()
		defer invLock.Unlock()

		var compartmentId uuid.UUID
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			c, err := p.GetByCharacterAndType(characterId)(inventoryType)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to get compartment by type [%d] for character [%d].", inventoryType, characterId)
				return err
			}
			compartmentId = c.Id()
			as := c.Assets()
			sort.Slice(as, func(i, j int) bool {
				return as[i].Slot() < as[j].Slot()
			})

			// Filter out assets with negative slot values
			var positiveSlotAssets []asset.Model[any]
			for _, a := range as {
				if a.Slot() >= 0 {
					positiveSlotAssets = append(positiveSlotAssets, a)
				}
			}

			// Merge combinable assets.
			for i := 0; i < len(positiveSlotAssets); i++ {
				for j := i + 1; j < len(positiveSlotAssets); j++ {
					if p.canMergeAssets(c.Type(), positiveSlotAssets[j], positiveSlotAssets[i], characterId) {
						err = p.Move(mb)(characterId)(inventoryType)(positiveSlotAssets[j].Slot())(positiveSlotAssets[i].Slot())
						if err != nil {
							p.l.WithError(err).Errorf("Unable to move assets [%d] and [%d] in compartment [%s].", positiveSlotAssets[i].Id(), positiveSlotAssets[j].Id(), c.Id())
							return err
						}
						c, err = p.GetByCharacterAndType(characterId)(inventoryType)
						if err != nil {
							p.l.WithError(err).Errorf("Unable to get compartment by type [%d] for character [%d].", inventoryType, characterId)
						}
						as = c.Assets()

						// Rebuild the positive slot assets list
						positiveSlotAssets = nil
						for _, a := range as {
							if a.Slot() >= 0 {
								positiveSlotAssets = append(positiveSlotAssets, a)
							}
						}

						sort.Slice(positiveSlotAssets, func(i, j int) bool {
							return positiveSlotAssets[i].Slot() < positiveSlotAssets[j].Slot()
						})
						j--
					}
				}
			}

			// Compact
			for i := 0; i < len(positiveSlotAssets); i++ {
				var nextFree int16
				nextFree, err = c.NextFreeSlot()
				if err != nil {
					continue
				}
				if positiveSlotAssets[i].Slot() >= nextFree {
					err = p.Move(mb)(characterId)(inventoryType)(positiveSlotAssets[i].Slot())(nextFree)
					if err != nil {
						p.l.WithError(err).Errorf("Unable to move assets [%d] in compartment [%s].", positiveSlotAssets[i].Id(), c.Id())
						return err
					}
					c, err = p.GetByCharacterAndType(characterId)(inventoryType)
					if err != nil {
						p.l.WithError(err).Errorf("Unable to get compartment by type [%d] for character [%d].", inventoryType, characterId)
					}
					as = c.Assets()

					// Rebuild the positive slot assets list
					positiveSlotAssets = nil
					for _, a := range as {
						if a.Slot() >= 0 {
							positiveSlotAssets = append(positiveSlotAssets, a)
						}
					}

					sort.Slice(positiveSlotAssets, func(i, j int) bool {
						return positiveSlotAssets[i].Slot() < positiveSlotAssets[j].Slot()
					})
				}
			}

			return nil
		})

		if txErr != nil {
			p.l.WithError(txErr).Errorf("Character [%d] unable to merge and compact assets in inventory [%d].", characterId, inventoryType)
			return txErr
		}

		// Emit the status event for successful completion
		err := mb.Put(compartment.EnvEventTopicStatus, MergeCompleteEventStatusProvider(compartmentId, characterId, inventoryType))
		if err != nil {
			p.l.WithError(err).Errorf("Unable to emit merge and compact complete event for character [%d], inventory [%d].", characterId, inventoryType)
			return err
		}

		p.l.Debugf("Character [%d] successfully merged and compacted assets in inventory [%d].", characterId, inventoryType)
		return nil
	}
}

func (p *Processor) CompactAndSort(mb *message.Buffer) func(characterId uint32, inventoryType inventory.Type) error {
	return func(characterId uint32, inventoryType inventory.Type) error {
		p.l.Debugf("Character [%d] attempting to compact and sort assets in inventory [%d].", characterId, inventoryType)

		invLock := LockRegistry().Get(characterId, inventoryType)
		invLock.Lock()
		defer invLock.Unlock()

		var compartmentId uuid.UUID
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			c, err := p.GetByCharacterAndType(characterId)(inventoryType)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to get compartment by type [%d] for character [%d].", inventoryType, characterId)
				return err
			}
			compartmentId = c.Id()
			as := c.Assets()

			// Filter out assets with negative slot values
			var positiveSlotAssets []asset.Model[any]
			for _, a := range as {
				if a.Slot() >= 0 {
					positiveSlotAssets = append(positiveSlotAssets, a)
				}
			}

			// Compact
			for i := 0; i < len(positiveSlotAssets); i++ {
				var nextFree int16
				nextFree, err = c.NextFreeSlot()
				if err != nil {
					continue
				}
				if positiveSlotAssets[i].Slot() >= nextFree {
					err = p.Move(mb)(characterId)(inventoryType)(positiveSlotAssets[i].Slot())(nextFree)
					if err != nil {
						p.l.WithError(err).Errorf("Unable to move assets [%d] in compartment [%s].", positiveSlotAssets[i].Id(), c.Id())
						return err
					}
					c, err = p.GetByCharacterAndType(characterId)(inventoryType)
					if err != nil {
						p.l.WithError(err).Errorf("Unable to get compartment by type [%d] for character [%d].", inventoryType, characterId)
					}
					as = c.Assets()

					// Rebuild the positive slot assets list
					positiveSlotAssets = nil
					for _, a := range as {
						if a.Slot() >= 0 {
							positiveSlotAssets = append(positiveSlotAssets, a)
						}
					}

					sort.Slice(positiveSlotAssets, func(i, j int) bool {
						return positiveSlotAssets[i].Slot() < positiveSlotAssets[j].Slot()
					})
				}
			}

			// Sorting assets
			for i := 0; i < len(positiveSlotAssets); i++ {
				minIdx := i
				for j := i + 1; j < len(positiveSlotAssets); j++ {
					if positiveSlotAssets[j].TemplateId() < positiveSlotAssets[minIdx].TemplateId() {
						minIdx = j
					}
				}
				if minIdx != i {
					err = p.Move(mb)(characterId)(inventoryType)(positiveSlotAssets[minIdx].Slot())(positiveSlotAssets[i].Slot())
					if err != nil {
						p.l.WithError(err).Errorf("Unable to move assets [%d] and [%d] in compartment [%s].", positiveSlotAssets[i].Id(), positiveSlotAssets[minIdx].Id(), c.Id())
						return err
					}
					c, err = p.GetByCharacterAndType(characterId)(inventoryType)
					if err != nil {
						p.l.WithError(err).Errorf("Unable to get compartment by type [%d] for character [%d].", inventoryType, characterId)
					}
					as = c.Assets()

					// Rebuild the positive slot assets list
					positiveSlotAssets = nil
					for _, a := range as {
						if a.Slot() >= 0 {
							positiveSlotAssets = append(positiveSlotAssets, a)
						}
					}

					sort.Slice(positiveSlotAssets, func(i, j int) bool {
						return positiveSlotAssets[i].Slot() < positiveSlotAssets[j].Slot()
					})
				}
			}

			return nil
		})

		if txErr != nil {
			p.l.WithError(txErr).Errorf("Character [%d] unable to compact and sort assets in inventory [%d].", characterId, inventoryType)
			return txErr
		}

		// Emit the status event for successful completion
		err := mb.Put(compartment.EnvEventTopicStatus, SortCompleteEventStatusProvider(compartmentId, characterId, inventoryType))
		if err != nil {
			p.l.WithError(err).Errorf("Unable to emit compact and sort complete event for character [%d], inventory [%d].", characterId, inventoryType)
			return err
		}

		p.l.Debugf("Character [%d] successfully compacted and sorted assets in inventory [%d].", characterId, inventoryType)
		return nil
	}
}
