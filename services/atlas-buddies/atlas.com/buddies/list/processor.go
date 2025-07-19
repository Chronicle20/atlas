package list

import (
	"atlas-buddies/buddy"
	"atlas-buddies/character"
	"atlas-buddies/database"
	"atlas-buddies/invite"
	"atlas-buddies/kafka/message"
	list2 "atlas-buddies/kafka/message/list"
	"atlas-buddies/kafka/producer"
	list3 "atlas-buddies/kafka/producer/list"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	WithTransaction(*gorm.DB) Processor
	ByCharacterIdProvider(characterId uint32) model.Provider[Model]
	GetByCharacterId(characterId uint32) (Model, error)
	Create(characterId uint32, capacity byte) (Model, error)
	DeleteAndEmit(characterId uint32, worldId byte) error
	Delete(mb *message.Buffer) func(characterId uint32, worldId byte) error
	RequestAddBuddyAndEmit(characterId uint32, worldId byte, targetId uint32, group string) error
	RequestAddBuddy(mb *message.Buffer) func(characterId uint32, worldId byte, targetId uint32, group string) error
	RequestDeleteBuddyAndEmit(characterId uint32, worldId byte, targetId uint32) error
	RequestDeleteBuddy(mb *message.Buffer) func(characterId uint32, worldId byte, targetId uint32) error
	AcceptInviteAndEmit(characterId uint32, worldId byte, targetId uint32) error
	AcceptInvite(mb *message.Buffer) func(characterId uint32, worldId byte, targetId uint32) error
	DeleteBuddyAndEmit(characterId uint32, worldId byte, targetId uint32) error
	DeleteBuddy(mb *message.Buffer) func(characterId uint32, worldId byte, targetId uint32) error
	UpdateBuddyChannelAndEmit(characterId uint32, worldId byte, channelId int8) error
	UpdateBuddyChannel(mb *message.Buffer) func(characterId uint32, worldId byte, channelId int8) error
	UpdateBuddyShopStatusAndEmit(characterId uint32, worldId byte, inShop bool) error
	UpdateBuddyShopStatus(mb *message.Buffer) func(characterId uint32, worldId byte, inShop bool) error
	// IncreaseCapacityAndEmit increases buddy list capacity and emits appropriate status events.
	// This method validates the new capacity and updates the database in a transaction.
	// On success, emits a CAPACITY_CHANGE event. On failure, emits an ERROR event.
	IncreaseCapacityAndEmit(characterId uint32, worldId byte, newCapacity byte) error
	// IncreaseCapacity is the pure business logic version that accepts a message buffer
	// for transactional event emission. Use this when you need to coordinate multiple
	// operations within a single transaction.
	IncreaseCapacity(mb *message.Buffer) func(characterId uint32, worldId byte, newCapacity byte) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
	p   producer.Provider
	cp  character.Processor
	ip  invite.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
		p:   producer.ProviderImpl(l)(ctx),
		cp:  character.NewProcessor(l, ctx),
		ip:  invite.NewProcessor(l, ctx),
	}
}

func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   p.l,
		ctx: p.ctx,
		db:  tx,
		t:   p.t,
		p:   p.p,
	}
}

func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[Model] {
	return model.Map(Make)(byCharacterIdEntityProvider(p.t.Id(), characterId)(p.db))
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) (Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *ProcessorImpl) Create(characterId uint32, capacity byte) (Model, error) {
	p.l.Debugf("Creating buddy list for character [%d] with a capacity of [%d].", characterId, capacity)
	m, err := create(p.db, p.t, characterId, capacity)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to create initial buddy list for character [%d].", characterId)
		return Model{}, err
	}
	return m, nil
}

func (p *ProcessorImpl) DeleteAndEmit(characterId uint32, worldId byte) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Delete(buf)(characterId, worldId)
	})
}

func (p *ProcessorImpl) Delete(mb *message.Buffer) func(characterId uint32, worldId byte) error {
	return func(characterId uint32, worldId byte) error {
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			bl, err := p.WithTransaction(tx).GetByCharacterId(characterId)
			if err != nil {
				return err
			}

			// Remove deleted character for all of their buddies.
			for _, b := range bl.Buddies() {
				err = removeBuddy(tx, p.t.Id(), b.CharacterId(), characterId)
				if err != nil {
					p.l.WithError(err).Errorf("Unable to remove buddy from buddy list for character [%d].", b.CharacterId())
					return err
				}

				_ = mb.Put(list2.EnvStatusEventTopic, list3.BuddyRemovedStatusEventProvider(b.CharacterId(), worldId, characterId))
			}
			return deleteEntityWithBuddies(tx, p.t.Id(), characterId)
		})
		if txErr != nil {
			return txErr
		}
		return nil
	}
}

func (p *ProcessorImpl) RequestAddBuddyAndEmit(characterId uint32, worldId byte, targetId uint32, group string) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.RequestAddBuddy(buf)(characterId, worldId, targetId, group)
	})
}

func (p *ProcessorImpl) RequestAddBuddy(mb *message.Buffer) func(characterId uint32, worldId byte, targetId uint32, group string) error {
	return func(characterId uint32, worldId byte, targetId uint32, group string) error {
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			tc, err := p.cp.GetById(targetId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to retrieve character [%d] information.", targetId)
				_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorCharacterNotFound))
				return err
			}

			if tc.GM() > 0 {
				p.l.Infof("Character [%d] attempting to buddy a GM.", characterId)
				_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorCannotBuddyGm))
				return errors.New("cannot buddy a gm")
			}

			cbl, err := p.WithTransaction(tx).GetByCharacterId(characterId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to retrieve buddy list for character [%d] attempting to add buddy.", characterId)
				_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
				return err
			}

			if byte(len(cbl.Buddies()))+1 > cbl.Capacity() {
				p.l.Infof("Buddy list for character [%d] is at capacity.", characterId)
				_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorListFull))
				return errors.New("buddy list is at capacity")
			}

			var found = false
			for _, b := range cbl.Buddies() {
				if b.CharacterId() == targetId {
					found = true
					break
				}
			}
			if found {
				p.l.Infof("Target [%d] is already on character [%d] buddy list.", targetId, characterId)
				_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorAlreadyBuddy))
				return errors.New("buddy already exists")
			}

			obl, err := p.WithTransaction(tx).GetByCharacterId(targetId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to retrieve buddy list for character [%d] being added as buddy.", targetId)
				_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
				return err
			}

			if byte(len(obl.Buddies()))+1 > obl.Capacity() {
				p.l.Infof("Buddy list for character [%d] is at capacity.", targetId)
				_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorListFull))
				return errors.New("buddy list is at capacity")
			}

			var mbe *buddy.Model
			for _, b := range obl.Buddies() {
				if b.CharacterId() == characterId {
					mbe = &b
					break
				}
			}
			if mbe != nil {
				p.l.Infof("Character [%d] is already on target characters [%d] buddy list.", characterId, targetId)
				err = addBuddy(tx, p.t.Id(), characterId, targetId, tc.Name(), group, false)
				if err != nil {
					p.l.WithError(err).Errorf("Unable to add buddy to buddy list for character [%d].", characterId)
					_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
					return err
				}

				_ = mb.Put(list2.EnvStatusEventTopic, list3.BuddyAddedStatusEventProvider(characterId, worldId, targetId, tc.Name(), -1, group))
				_ = mb.Put(list2.EnvStatusEventTopic, list3.BuddyAddedStatusEventProvider(targetId, worldId, characterId, mbe.Name(), mbe.ChannelId(), mbe.Group()))
				// TODO need to trigger a channel request for target.
				return nil
			}

			// soft allocate buddy for character
			err = addPendingBuddy(tx, p.t.Id(), characterId, targetId, tc.Name(), group)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to add buddy to buddy list for character [%d].", characterId)
				_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
				return err
			}
			err = p.ip.Create(characterId, worldId, targetId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to create invite for character [%d] to buddy character [%d].", characterId, targetId)
				_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
				return err
			}
			_ = mb.Put(list2.EnvStatusEventTopic, list3.BuddyAddedStatusEventProvider(characterId, worldId, targetId, tc.Name(), -1, group))
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to add buddy to buddy list for character [%d].", characterId)
			return nil
		}
		return nil
	}
}

func (p *ProcessorImpl) RequestDeleteBuddyAndEmit(characterId uint32, worldId byte, targetId uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.RequestDeleteBuddy(buf)(characterId, worldId, targetId)
	})
}

func (p *ProcessorImpl) RequestDeleteBuddy(mb *message.Buffer) func(characterId uint32, worldId byte, targetId uint32) error {
	return func(characterId uint32, worldId byte, targetId uint32) error {
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			cbl, err := p.WithTransaction(tx).GetByCharacterId(characterId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to retrieve buddy list for character [%d] attempting to add buddy.", characterId)
				_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
				return err
			}

			var found = false
			for _, b := range cbl.Buddies() {
				if b.CharacterId() == targetId {
					found = true
					break
				}
			}
			if !found {
				p.l.Debugf("Target [%d] is not on character [%d] buddy list. This could be an invite rejection.", targetId, characterId)
				err = p.ip.Reject(characterId, worldId, targetId)
				if err != nil {
					p.l.WithError(err).Errorf("Unable to reject invite for character [%d] to buddy character [%d].", characterId, targetId)
					_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
					return err
				}
				return nil
			}

			err = removeBuddy(tx, p.t.Id(), characterId, targetId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to remove buddy from buddy list for character [%d].", characterId)
				_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
				return err
			}

			var update bool
			update, err = updateBuddyChannel(tx, p.t.Id(), characterId, targetId, -1)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to update character [%d] channel to [%d] in [%d] buddy list.", characterId, -1, targetId)
				_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
				return err
			}

			_ = mb.Put(list2.EnvStatusEventTopic, list3.BuddyRemovedStatusEventProvider(characterId, worldId, targetId))

			if update {
				_ = mb.Put(list2.EnvStatusEventTopic, list3.BuddyChannelChangeStatusEventProvider(targetId, worldId, characterId, -1))
			}
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to remove buddy from buddy list for character [%d].", characterId)
			return nil
		}
		return nil
	}
}

func (p *ProcessorImpl) AcceptInviteAndEmit(characterId uint32, worldId byte, targetId uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.AcceptInvite(buf)(characterId, worldId, targetId)
	})
}

func (p *ProcessorImpl) AcceptInvite(mb *message.Buffer) func(characterId uint32, worldId byte, targetId uint32) error {
	return func(characterId uint32, worldId byte, targetId uint32) error {
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			cbl, err := p.WithTransaction(tx).GetByCharacterId(characterId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to retrieve buddy list for character [%d] attempting to add buddy.", characterId)
				_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
				return err
			}

			if byte(len(cbl.Buddies()))+1 > cbl.Capacity() {
				p.l.Infof("Buddy list for character [%d] is at capacity.", characterId)
				_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorListFull))
				return errors.New("buddy list is at capacity")
			}

			var found = false
			for _, b := range cbl.Buddies() {
				if b.CharacterId() == targetId {
					found = true
					break
				}
			}
			if found {
				p.l.Infof("Target [%d] is already on character [%d] buddy list.", targetId, characterId)
				_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorAlreadyBuddy))
				return errors.New("buddy already exists")
			}

			obl, err := p.WithTransaction(tx).GetByCharacterId(targetId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to retrieve buddy list for character [%d] attempting to add buddy.", characterId)
				_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
				return err
			}
			var ob buddy.Model
			for _, b := range obl.Buddies() {
				if b.CharacterId() == characterId {
					ob = b
				}
			}

			c, err := p.cp.GetById(characterId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to retrieve character [%d] information.", characterId)
				_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
				return err
			}

			oc, err := p.cp.GetById(targetId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to retrieve character [%d] information.", targetId)
				_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorCharacterNotFound))
				return err
			}

			err = removeBuddy(tx, p.t.Id(), targetId, characterId)
			if err != nil {
				return err
			}

			err = addBuddy(tx, p.t.Id(), characterId, targetId, oc.Name(), "Default Group", false)
			if err != nil {
				return err
			}

			err = addBuddy(tx, p.t.Id(), targetId, characterId, c.Name(), ob.Group(), false)
			if err != nil {
				return err
			}

			_ = mb.Put(list2.EnvStatusEventTopic, list3.BuddyAddedStatusEventProvider(characterId, worldId, targetId, oc.Name(), -1, "Default Group"))
			// TODO need to trigger a channel request for target.
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to add buddy to buddy list for character [%d].", characterId)
			return nil
		}
		return nil
	}
}

func (p *ProcessorImpl) DeleteBuddyAndEmit(characterId uint32, worldId byte, targetId uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.DeleteBuddy(buf)(characterId, worldId, targetId)
	})
}

func (p *ProcessorImpl) DeleteBuddy(mb *message.Buffer) func(characterId uint32, worldId byte, targetId uint32) error {
	return func(characterId uint32, worldId byte, targetId uint32) error {
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			err := removeBuddy(tx, p.t.Id(), characterId, targetId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to remove buddy from buddy list for character [%d].", characterId)
				return err
			}
			var update bool
			update, err = updateBuddyChannel(tx, p.t.Id(), characterId, targetId, -1)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to update character [%d] channel to [%d] in [%d] buddy list.", characterId, -1, targetId)
				return err
			}

			_ = mb.Put(list2.EnvStatusEventTopic, list3.BuddyRemovedStatusEventProvider(characterId, worldId, targetId))

			if update {
				_ = mb.Put(list2.EnvStatusEventTopic, list3.BuddyChannelChangeStatusEventProvider(targetId, worldId, characterId, -1))
			}
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to remove buddy from buddy list for character [%d].", characterId)
			return txErr
		}
		return nil
	}
}

func (p *ProcessorImpl) UpdateBuddyChannelAndEmit(characterId uint32, worldId byte, channelId int8) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.UpdateBuddyChannel(buf)(characterId, worldId, channelId)
	})
}

func (p *ProcessorImpl) UpdateBuddyChannel(mb *message.Buffer) func(characterId uint32, worldId byte, channelId int8) error {
	return func(characterId uint32, worldId byte, channelId int8) error {
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			bl, err := byCharacterIdEntityProvider(p.t.Id(), characterId)(tx)()
			if err != nil {
				p.l.WithError(err).Errorf("Unable to locate buddy list for character [%d].", characterId)
				return err
			}
			for _, b := range bl.Buddies {
				var update bool
				update, err = updateBuddyChannel(tx, p.t.Id(), characterId, b.CharacterId, channelId)
				if err != nil {
					p.l.WithError(err).Errorf("Unable to update character [%d] channel to [%d] in [%d] buddy list.", characterId, channelId, b.CharacterId)
					return err
				}

				if update {
					_ = mb.Put(list2.EnvStatusEventTopic, list3.BuddyChannelChangeStatusEventProvider(b.CharacterId, worldId, characterId, channelId))
				}
			}
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to update buddy channel for character [%d].", characterId)
			return txErr
		}
		return nil
	}
}

func (p *ProcessorImpl) UpdateBuddyShopStatusAndEmit(characterId uint32, worldId byte, inShop bool) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.UpdateBuddyShopStatus(buf)(characterId, worldId, inShop)
	})
}

func (p *ProcessorImpl) UpdateBuddyShopStatus(mb *message.Buffer) func(characterId uint32, worldId byte, inShop bool) error {
	return func(characterId uint32, worldId byte, inShop bool) error {
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			bl, err := byCharacterIdEntityProvider(p.t.Id(), characterId)(tx)()
			if err != nil {
				p.l.WithError(err).Errorf("Unable to locate buddy list for character [%d].", characterId)
				return err
			}
			for _, b := range bl.Buddies {
				var update bool
				update, err = updateBuddyShopStatus(tx, p.t.Id(), characterId, b.CharacterId, inShop)
				if err != nil {
					p.l.WithError(err).Errorf("Unable to update character [%d] shop status to [%t] in [%d] buddy list.", characterId, inShop, b.CharacterId)
					return err
				}

				if update {
					tbl, err := byCharacterIdEntityProvider(p.t.Id(), b.CharacterId)(tx)()
					if err != nil {
						return err
					}
					var tbe *buddy.Entity
					for _, pbe := range tbl.Buddies {
						if pbe.CharacterId == characterId {
							tbe = &pbe
						}
					}
					if tbe == nil {
						continue
					}

					_ = mb.Put(list2.EnvStatusEventTopic, list3.BuddyUpdatedStatusEventProvider(b.CharacterId, worldId, tbe.CharacterId, tbe.Group, tbe.CharacterName, b.ChannelId, inShop))
				}
			}
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Unable to update buddy shop status for character [%d].", characterId)
			return txErr
		}
		return nil
	}
}

// IncreaseCapacityAndEmit increases the buddy list capacity for a character and emits status events.
// This method handles the complete workflow: validation, database update, and event emission.
//
// Parameters:
//   - characterId: ID of the character whose capacity should be increased
//   - worldId: ID of the world the character is in (for event context)
//   - newCapacity: The new capacity value (must be > current capacity)
//
// Returns:
//   - error: nil on success, error on failure
//
// Events Emitted:
//   - CAPACITY_CHANGE: On successful capacity increase with new capacity value
//   - ERROR with INVALID_CAPACITY: If newCapacity <= currentCapacity
//   - ERROR with CHARACTER_NOT_FOUND: If character's buddy list doesn't exist
//   - ERROR with UNKNOWN_ERROR: For unexpected database errors
//
// Validation:
//   - Character must have an existing buddy list
//   - New capacity must be strictly greater than current capacity
//   - All operations are performed within a database transaction
func (p *ProcessorImpl) IncreaseCapacityAndEmit(characterId uint32, worldId byte, newCapacity byte) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.IncreaseCapacity(buf)(characterId, worldId, newCapacity)
	})
}

// IncreaseCapacity returns a curried function that increases buddy list capacity within a transaction.
// This is the pure business logic version that works with a message buffer for event coordination.
//
// Parameters:
//   - mb: Message buffer for accumulating events within the transaction
//
// Returns:
//   - A function that takes (characterId, worldId, newCapacity) and returns an error
//
// Implementation Details:
//   - Retrieves current buddy list to validate existing capacity
//   - Validates newCapacity > currentCapacity (emits INVALID_CAPACITY error if not)
//   - Updates capacity using administrator function
//   - Emits CAPACITY_CHANGE event on success
//   - All operations are wrapped in a database transaction
//   - Follows the Atlas pattern of pure functions with message buffer coordination
func (p *ProcessorImpl) IncreaseCapacity(mb *message.Buffer) func(characterId uint32, worldId byte, newCapacity byte) error {
	return func(characterId uint32, worldId byte, newCapacity byte) error {
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			// Get current buddy list to validate capacity
			bl, err := p.WithTransaction(tx).GetByCharacterId(characterId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to retrieve buddy list for character [%d] to increase capacity.", characterId)
				_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorCharacterNotFound))
				return err
			}

			// Validate that new capacity is greater than current capacity
			if newCapacity <= bl.Capacity() {
				p.l.Debugf("Invalid capacity change attempt for character [%d]: new capacity [%d] must be greater than current capacity [%d].", characterId, newCapacity, bl.Capacity())
				_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorInvalidCapacity))
				return errors.New("new capacity must be greater than current capacity")
			}

			// Update the capacity using administrator function
			err = updateCapacity(tx, p.t.Id(), characterId, newCapacity)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to update capacity for character [%d] to [%d].", characterId, newCapacity)
				_ = mb.Put(list2.EnvStatusEventTopic, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
				return err
			}

			// Emit success event
			_ = mb.Put(list2.EnvStatusEventTopic, list3.BuddyCapacityChangeStatusEventProvider(characterId, worldId, newCapacity))
			p.l.Debugf("Successfully increased buddy list capacity for character [%d] to [%d].", characterId, newCapacity)
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Transaction failed while increasing capacity for character [%d].", characterId)
			return txErr
		}
		return nil
	}
}
