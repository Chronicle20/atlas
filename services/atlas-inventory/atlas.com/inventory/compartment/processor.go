package compartment

import (
	"atlas-inventory/asset"
	"atlas-inventory/drop"
	"atlas-inventory/equipment"
	"atlas-inventory/kafka/message"
	"atlas-inventory/kafka/message/compartment"
	"atlas-inventory/kafka/producer"
	compartment2 "atlas-inventory/kafka/producer/compartment"
	model2 "atlas-inventory/model"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas-constants/item"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
	"math"
	"time"

	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor struct {
	l                     logrus.FieldLogger
	ctx                   context.Context
	db                    *gorm.DB
	assetProcessor        *asset.Processor
	dropProcessor         *drop.Processor
	equipmentProcessor    *equipment.Processor
	producer              producer.Provider
	GetById               func(id uuid.UUID) (Model, error)
	GetByCharacterId      func(characterId uint32) ([]Model, error)
	GetByCharacterAndType func(characterId uint32) func(inventoryType inventory.Type) (Model, error)
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *Processor {
	p := &Processor{
		l:                  l,
		ctx:                ctx,
		db:                 db,
		assetProcessor:     asset.NewProcessor(l, ctx, db),
		dropProcessor:      drop.NewProcessor(l, ctx),
		equipmentProcessor: equipment.NewProcessor(l, ctx),
	}
	p.producer = producer.ProviderImpl(l)(ctx)
	p.GetById = model2.CollapseProvider(p.ByIdProvider)
	p.GetByCharacterId = model2.CollapseProvider(p.ByCharacterIdProvider)
	p.GetByCharacterAndType = model.Compose(model2.CollapseProvider, p.ByCharacterAndTypeProvider)
	return p
}

func (p *Processor) WithTransaction(db *gorm.DB) *Processor {
	return &Processor{
		l:                     p.l,
		ctx:                   p.ctx,
		db:                    db,
		assetProcessor:        p.assetProcessor,
		dropProcessor:         p.dropProcessor,
		equipmentProcessor:    p.equipmentProcessor,
		producer:              p.producer,
		GetById:               p.GetById,
		GetByCharacterId:      p.GetByCharacterId,
		GetByCharacterAndType: p.GetByCharacterAndType,
	}
}

func (p *Processor) ByIdProvider(id uuid.UUID) model.Provider[Model] {
	t := tenant.MustFromContext(p.ctx)
	cs, err := model.Map(Make)(getById(t.Id(), id)(p.db))()
	if err != nil {
		return model.ErrorProvider[Model](err)
	}
	return model.Map(p.DecorateAsset)(model.FixedProvider(cs))
}

func (p *Processor) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	t := tenant.MustFromContext(p.ctx)
	cs, err := model.SliceMap(Make)(getByCharacter(t.Id(), characterId)(p.db))(model.ParallelMap())()
	if err != nil {
		return model.ErrorProvider[[]Model](err)
	}
	return model.SliceMap(p.DecorateAsset)(model.FixedProvider(cs))(model.ParallelMap())
}

func (p *Processor) ByCharacterAndTypeProvider(characterId uint32) func(inventoryType inventory.Type) model.Provider[Model] {
	return func(inventoryType inventory.Type) model.Provider[Model] {
		t := tenant.MustFromContext(p.ctx)
		cs, err := model.Map(Make)(getByCharacterAndType(t.Id(), characterId, inventoryType)(p.db))()
		if err != nil {
			return model.ErrorProvider[Model](err)
		}
		return model.Map(p.DecorateAsset)(model.FixedProvider(cs))
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
		t := tenant.MustFromContext(p.ctx)
		p.l.Debugf("Attempting to create compartment of type [%d] for character [%d] with capacity [%d].", inventoryType, characterId, capacity)
		var c Model
		txErr := p.db.Transaction(func(tx *gorm.DB) error {
			var err error
			c, err = create(tx, t.Id(), characterId, inventoryType, capacity)
			if err != nil {
				return err
			}
			return mb.Put(compartment.EnvEventTopicStatus, compartment2.CreatedEventStatusProvider(c.Id(), characterId, c.Type(), c.Capacity()))
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
		t := tenant.MustFromContext(p.ctx)
		txErr := p.db.Transaction(func(tx *gorm.DB) error {
			err := model.ForEachSlice(model.FixedProvider(c.Assets()), p.assetProcessor.WithTransaction(tx).Delete(mb)(c.CharacterId(), c.Id()))
			if err != nil {
				return err
			}
			err = deleteById(tx, t.Id(), c.Id())
			if err != nil {
				return err
			}
			return mb.Put(compartment.EnvEventTopicStatus, compartment2.DeletedEventStatusProvider(c.Id(), c.CharacterId()))
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
				txErr := p.db.Transaction(func(tx *gorm.DB) error {
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
					actualDestination, err := p.equipmentProcessor.DestinationSlotProvider(destination)(a1.TemplateId())()
					if err != nil {
						p.l.WithError(err).Errorf("Unable to determine actual destination for item being equipped.")
						return err
					}
					err = p.assetProcessor.WithTransaction(tx).UpdateSlot(mb)(characterId, c.Id(), a1.Id(), assetProvider(actualDestination), model.FixedProvider(temporarySlot()))
					if err != nil {
						p.l.WithError(err).Errorf("Unable to update asset slot from [%d] to [%d]. Character [%d]. Compartment [%d].", actualDestination, temporarySlot(), characterId, c.Id())
						return err
					}
					err = p.assetProcessor.WithTransaction(tx).UpdateSlot(mb)(characterId, c.Id(), a1.Id(), model.FixedProvider(a1), model.FixedProvider(actualDestination))
					if err != nil {
						p.l.WithError(err).Errorf("Unable to update asset slot from [%d] to [%d]. Character [%d]. Compartment [%d].", a1.Slot(), actualDestination, characterId, c.Id())
						return err
					}
					err = p.assetProcessor.WithTransaction(tx).UpdateSlot(mb)(characterId, c.Id(), a1.Id(), assetProvider(temporarySlot()), model.FixedProvider(source))
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
						err = p.assetProcessor.WithTransaction(tx).UpdateSlot(mb)(characterId, c.Id(), a1.Id(), assetProvider(int16(ps.Position)), model.FixedProvider(nfs))
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
							if item.GetClassification(item.Id(ta.TemplateId())) == item.Classification(104) {
								var nfs int16
								nfs, err = c.NextFreeSlot()
								if err != nil {
									p.l.WithError(err).Errorf("No free slots for top.")
									return err
								}
								err = p.assetProcessor.WithTransaction(tx).UpdateSlot(mb)(characterId, c.Id(), a1.Id(), model.FixedProvider(ta), model.FixedProvider(nfs))
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
					p.l.Debugf("Unable to equip item in slot [%d] to [%d] for character [%d].", source, destination, characterId)
				}
				p.l.Debugf("Character [%d] equipped item [%d] in slot [%d].", characterId, a1.TemplateId(), destination)
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

				var a1 asset.Model[any]
				txErr := p.db.Transaction(func(tx *gorm.DB) error {
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
					err = p.assetProcessor.WithTransaction(tx).UpdateSlot(mb)(characterId, c.Id(), a1.Id(), assetProvider(source), fsp)
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
	return message.Emit(p.producer)(model.Flip(model.Flip(model.Flip(model.Flip(p.Move)(characterId))(inventoryType))(source))(destination))
}

func (p *Processor) Move(mb *message.Buffer) func(characterId uint32) func(inventoryType inventory.Type) func(source int16) func(destination int16) error {
	t := tenant.MustFromContext(p.ctx)
	return func(characterId uint32) func(inventoryType inventory.Type) func(source int16) func(destination int16) error {
		return func(inventoryType inventory.Type) func(source int16) func(destination int16) error {
			return func(source int16) func(destination int16) error {
				return func(destination int16) error {
					p.l.Debugf("Attempting to move asset in slot [%d] to [%d] for character [%d].", source, destination, characterId)
					invLock := LockRegistry().Get(characterId, inventoryType)
					invLock.Lock()
					defer invLock.Unlock()

					var a1 asset.Model[any]
					txErr := p.db.Transaction(func(tx *gorm.DB) error {
						var c Model
						var err error
						c, err = p.WithTransaction(tx).GetByCharacterAndType(characterId)(inventoryType)
						if err != nil {
							p.l.WithError(err).Errorf("Unable to get compartment by type [%d] for character [%d].", inventoryType, characterId)
							return err
						}

						assetProvider := p.assetProcessor.WithTransaction(tx).BySlotProvider(c.Id())
						a1, err = assetProvider(source)()
						if err != nil {
							p.l.WithError(err).Errorf("Unable to get asset in compartment [%d] by slot [%d].", c.Id(), source)
							return err
						}
						p.l.Debugf("Character [%d] is attempting to move asset [%d].", characterId, a1.TemplateId())

						err = p.assetProcessor.WithTransaction(tx).UpdateSlot(mb)(characterId, c.Id(), a1.Id(), assetProvider(destination), model.FixedProvider(temporarySlot()))
						if err != nil {
							p.l.WithError(err).Errorf("Unable to update asset slot from [%d] to [%d]. Character [%d]. Compartment [%d].", destination, temporarySlot(), characterId, c.Id())
							return err
						}
						err = p.assetProcessor.WithTransaction(tx).UpdateSlot(mb)(characterId, c.Id(), a1.Id(), model.FixedProvider(a1), model.FixedProvider(destination))
						if err != nil {
							p.l.WithError(err).Errorf("Unable to update asset slot from [%d] to [%d]. Character [%d]. Compartment [%d].", a1.Slot(), destination, characterId, c.Id())
							return err
						}
						err = p.assetProcessor.WithTransaction(tx).UpdateSlot(mb)(characterId, c.Id(), a1.Id(), assetProvider(temporarySlot()), model.FixedProvider(source))
						if err != nil {
							p.l.WithError(err).Errorf("Unable to update asset slot from [%d] to [%d]. Character [%d]. Compartment [%d].", temporarySlot(), source, characterId, c.Id())
							return err
						}

						GetReservationRegistry().SwapReservation(t, characterId, inventoryType, source, destination)
						return nil
					})
					if txErr != nil {
						p.l.Debugf("Unable to move asset in slot [%d] to [%d] for character [%d].", source, destination, characterId)
					}
					p.l.Debugf("Character [%d] moved asset [%d] to slot [%d].", characterId, a1.TemplateId(), destination)
					return nil
				}
			}
		}
	}
}

func (p *Processor) IncreaseCapacityAndEmit(characterId uint32, inventoryType inventory.Type, amount uint32) error {
	return message.Emit(p.producer)(model.Flip(model.Flip(model.Flip(p.IncreaseCapacity)(characterId))(inventoryType))(amount))
}

func (p *Processor) IncreaseCapacity(mb *message.Buffer) func(characterId uint32) func(inventoryType inventory.Type) func(amount uint32) error {
	t := tenant.MustFromContext(p.ctx)
	return func(characterId uint32) func(inventoryType inventory.Type) func(amount uint32) error {
		return func(inventoryType inventory.Type) func(amount uint32) error {
			return func(amount uint32) error {
				p.l.Debugf("Character [%d] attempting to change compartment capacity by [%d]. Type [%d].", characterId, amount, inventoryType)
				invLock := LockRegistry().Get(characterId, inventoryType)
				invLock.Lock()
				defer invLock.Unlock()

				var capacity uint32
				txErr := p.db.Transaction(func(tx *gorm.DB) error {
					c, err := p.GetByCharacterAndType(characterId)(inventoryType)
					if err != nil {
						return err
					}
					capacity = uint32(math.Min(96, float64(c.Capacity()+amount)))
					_, err = updateCapacity(tx, t.Id(), characterId, int8(inventoryType), capacity)
					if err != nil {
						return err
					}
					return mb.Put(compartment.EnvEventTopicStatus, compartment2.CapacityChangedEventStatusProvider(c.Id(), characterId, inventoryType, capacity))
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

		t := tenant.MustFromContext(p.ctx)
		invLock := LockRegistry().Get(characterId, inventoryType)
		invLock.Lock()
		defer invLock.Unlock()

		var a asset.Model[any]
		txErr := p.db.Transaction(func(tx *gorm.DB) error {
			c, err := p.GetByCharacterAndType(characterId)(inventoryType)
			if err != nil {
				return err
			}
			a, err = p.assetProcessor.WithTransaction(tx).GetBySlot(c.Id(), source)
			if err != nil {
				return err
			}
			reservedQty := GetReservationRegistry().GetReservedQuantity(t, characterId, inventoryType, source)
			initialQty := a.Quantity() - reservedQty

			if initialQty < uint32(quantity) {
				return errors.New("cannot drop more than what is owned")
			}
			if initialQty == uint32(quantity) {
				err = p.assetProcessor.WithTransaction(tx).Delete(mb)(characterId, c.Id())(a)
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
		t := tenant.MustFromContext(p.ctx)
		p.l.Debugf("Character [%d] attempting to reserve [%d] inventory [%d] reservation [%s].", characterId, len(reservationRequests), inventoryType, transactionId.String())
		invLock := LockRegistry().Get(characterId, inventoryType)
		invLock.Lock()
		defer invLock.Unlock()

		txErr := p.db.Transaction(func(tx *gorm.DB) error {
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
				currentReservedQty := GetReservationRegistry().GetReservedQuantity(t, characterId, inventoryType, request.Slot)
				if a.Quantity()-currentReservedQty < uint32(request.Quantity) {
					return errors.New("cannot reserve more than what is owned")
				}
				_, err = GetReservationRegistry().AddReservation(t, transactionId, characterId, inventoryType, request.Slot, request.ItemId, uint32(request.Quantity), time.Second*time.Duration(30))
				if err != nil {
					return err
				}
				return mb.Put(compartment.EnvEventTopicStatus, compartment2.ReservedEventStatusProvider(c.Id(), characterId, request.ItemId, request.Slot, uint32(request.Quantity), transactionId))
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
		t := tenant.MustFromContext(p.ctx)
		p.l.Debugf("Character [%d] attempting to cancel inventory [%d] reservation [%s].", characterId, inventoryType, transactionId.String())
		invLock := LockRegistry().Get(characterId, inventoryType)
		invLock.Lock()
		defer invLock.Unlock()

		c, err := p.GetByCharacterAndType(characterId)(inventoryType)
		if err != nil {
			return err
		}

		res, err := GetReservationRegistry().RemoveReservation(t, transactionId, characterId, inventoryType, slot)
		if err != nil {
			return nil
		}
		return mb.Put(compartment.EnvEventTopicStatus, compartment2.ReservationCancelledEventStatusProvider(c.Id(), characterId, res.ItemId(), slot, res.Quantity()))
	}
}

func (p *Processor) ConsumeAssetAndEmit(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return p.ConsumeAsset(buf)(characterId, inventoryType, transactionId, slot)
	})
}

func (p *Processor) ConsumeAsset(mb *message.Buffer) func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
	return func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
		t := tenant.MustFromContext(p.ctx)
		p.l.Debugf("Character [%d] attempting to consume asset in inventory [%d] slot [%d]. Transaction [%d].", characterId, inventoryType, transactionId, slot)
		invLock := LockRegistry().Get(characterId, inventoryType)
		invLock.Lock()
		defer invLock.Unlock()

		res, err := GetReservationRegistry().RemoveReservation(t, transactionId, characterId, inventoryType, slot)
		if err != nil {
			return nil
		}

		var a asset.Model[any]
		txErr := p.db.Transaction(func(tx *gorm.DB) error {
			var c Model
			c, err = p.WithTransaction(tx).GetByCharacterAndType(characterId)(inventoryType)
			if err != nil {
				return err
			}
			a, err = p.assetProcessor.WithTransaction(tx).GetBySlot(c.Id(), slot)
			if err != nil {
				return err
			}
			reservedQty := GetReservationRegistry().GetReservedQuantity(t, characterId, inventoryType, slot)
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

func (p *Processor) DestroyAssetAndEmit(characterId uint32, inventoryType inventory.Type, slot int16) error {
	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return p.DestroyAsset(buf)(characterId, inventoryType, slot)
	})
}

func (p *Processor) DestroyAsset(mb *message.Buffer) func(characterId uint32, inventoryType inventory.Type, slot int16) error {
	return func(characterId uint32, inventoryType inventory.Type, slot int16) error {
		p.l.Debugf("Character [%d] attempting to destroy asset in inventory [%d] slot [%d].", characterId, inventoryType, slot)
		invLock := LockRegistry().Get(characterId, inventoryType)
		invLock.Lock()
		defer invLock.Unlock()

		var a asset.Model[any]
		txErr := p.db.Transaction(func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetByCharacterAndType(characterId)(inventoryType)
			if err != nil {
				return err
			}
			a, err = p.assetProcessor.WithTransaction(tx).GetBySlot(c.Id(), slot)
			if err != nil {
				return err
			}
			err = p.assetProcessor.Delete(mb)(characterId, c.Id())(a)
			if err != nil {
				return err
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
		txErr := p.db.Transaction(func(tx *gorm.DB) error {
			c, err := p.WithTransaction(tx).GetByCharacterAndType(characterId)(inventoryType)
			if err != nil {
				return err
			}
			nfs, err := c.NextFreeSlot()
			if err != nil {
				return err
			}
			err = p.assetProcessor.WithTransaction(tx).Create(mb)(characterId, c.Id(), templateId, nfs, quantity, expiration, ownerId, flag, rechargeable)
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
