package list

import (
	"atlas-buddies/buddy"
	"atlas-buddies/character"
	"atlas-buddies/invite"
	"atlas-buddies/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func byCharacterIdProvider(_ logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) model.Provider[Model] {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) model.Provider[Model] {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32) model.Provider[Model] {
			return func(characterId uint32) model.Provider[Model] {
				return model.Map(Make)(byCharacterIdEntityProvider(t.Id(), characterId)(db))
			}
		}
	}
}

func GetByCharacterId(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32) (Model, error) {
		return func(db *gorm.DB) func(characterId uint32) (Model, error) {
			return func(characterId uint32) (Model, error) {
				return byCharacterIdProvider(l)(ctx)(db)(characterId)()
			}
		}
	}
}

func Create(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, capacity byte) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, capacity byte) (Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, capacity byte) (Model, error) {
			return func(characterId uint32, capacity byte) (Model, error) {
				l.Debugf("Creating buddy list for character [%d] with a capacity of [%d].", characterId, capacity)
				m, err := create(db, t, characterId, capacity)
				if err != nil {
					l.WithError(err).Errorf("Unable to create initial buddy list for character [%d].", characterId)
					return Model{}, err
				}
				return m, nil
			}
		}
	}
}

func RequestAdd(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32, group string) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32, group string) error {
		t := tenant.MustFromContext(ctx)
		statusEventProducer := producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)
		return func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32, group string) error {
			return func(characterId uint32, worldId byte, targetId uint32, group string) error {
				return db.Transaction(func(tx *gorm.DB) error {
					tc, err := character.GetById(l)(ctx)(targetId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve character [%d] information.", targetId)
						err = statusEventProducer(errorStatusEventProvider(characterId, worldId, StatusEventErrorCharacterNotFound))
						return err
					}

					if tc.GM() > 0 {
						l.Infof("Character [%d] attempting to buddy a GM.", characterId)
						err = statusEventProducer(errorStatusEventProvider(characterId, worldId, StatusEventErrorCannotBuddyGm))
						return nil
					}

					cbl, err := GetByCharacterId(l)(ctx)(tx)(characterId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve buddy list for character [%d] attempting to add buddy.", characterId)
						err = statusEventProducer(errorStatusEventProvider(characterId, worldId, StatusEventErrorUnknownError))
						return err
					}

					if byte(len(cbl.Buddies()))+1 > cbl.Capacity() {
						l.Infof("Buddy list for character [%d] is at capacity.", characterId)
						err = statusEventProducer(errorStatusEventProvider(characterId, worldId, StatusEventErrorListFull))
						return err
					}

					var found = false
					for _, b := range cbl.Buddies() {
						if b.CharacterId() == targetId {
							found = true
							break
						}
					}
					if found {
						l.Infof("Target [%d] is already on character [%d] buddy list.", targetId, characterId)
						err = statusEventProducer(errorStatusEventProvider(characterId, worldId, StatusEventErrorAlreadyBuddy))
						return err
					}

					obl, err := GetByCharacterId(l)(ctx)(tx)(targetId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve buddy list for character [%d] being added as buddy.", targetId)
						err = statusEventProducer(errorStatusEventProvider(characterId, worldId, StatusEventErrorUnknownError))
						return err
					}

					if byte(len(obl.Buddies()))+1 > obl.Capacity() {
						l.Infof("Buddy list for character [%d] is at capacity.", targetId)
						err = statusEventProducer(errorStatusEventProvider(characterId, worldId, StatusEventErrorOtherListFull))
						return err
					}

					// soft allocate buddy for character
					err = addPendingBuddy(tx, t.Id(), characterId, targetId, tc.Name(), group)
					if err != nil {
						l.WithError(err).Errorf("Unable to add buddy to buddy list for character [%d].", characterId)
						err = statusEventProducer(errorStatusEventProvider(characterId, worldId, StatusEventErrorUnknownError))
						return err
					}
					err = invite.Create(l)(ctx)(characterId, worldId, targetId)
					if err != nil {
						err = statusEventProducer(errorStatusEventProvider(characterId, worldId, StatusEventErrorUnknownError))
						return err
					}
					err = statusEventProducer(buddyAddedStatusEventProvider(characterId, worldId, targetId, tc.Name(), -1, group))
					if err != nil {
						l.WithError(err).Errorf("Unable to identify to character [%d] that the invite was accepted.", characterId)
					}
					return nil
				})
			}
		}
	}
}

func RequestDelete(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
		t := tenant.MustFromContext(ctx)
		statusEventProducer := producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)
		return func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
			return func(characterId uint32, worldId byte, targetId uint32) error {
				return db.Transaction(func(tx *gorm.DB) error {
					cbl, err := GetByCharacterId(l)(ctx)(tx)(characterId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve buddy list for character [%d] attempting to add buddy.", characterId)
						err = statusEventProducer(errorStatusEventProvider(characterId, worldId, StatusEventErrorUnknownError))
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
						l.Debugf("Target [%d] is not on character [%d] buddy list. This could be an invite rejection.", targetId, characterId)
						err = invite.Reject(l)(ctx)(characterId, worldId, targetId)
						if err != nil {
							err = statusEventProducer(errorStatusEventProvider(characterId, worldId, StatusEventErrorUnknownError))
							return err
						}
						return nil
					}

					err = removeBuddy(tx, t.Id(), characterId, targetId)
					if err != nil {
						l.WithError(err).Errorf("Unable to remove buddy from buddy list for character [%d].", characterId)
						err = statusEventProducer(errorStatusEventProvider(characterId, worldId, StatusEventErrorUnknownError))
						return err
					}

					var update bool
					update, err = updateBuddyChannel(tx, t.Id(), characterId, targetId, -1)
					if err != nil {
						l.WithError(err).Errorf("Unable to update character [%d] channel to [%d] in [%d] buddy list.", characterId, -1, targetId)
						err = statusEventProducer(errorStatusEventProvider(characterId, worldId, StatusEventErrorUnknownError))
						return err
					}

					err = statusEventProducer(buddyRemovedStatusEventProvider(characterId, worldId, targetId))
					if err != nil {
						l.WithError(err).Errorf("Unable to inform [%d] the buddy [%d] was removed.", characterId, targetId)
					}

					if update {
						err = statusEventProducer(buddyChannelChangeStatusEventProvider(targetId, worldId, characterId, -1))
						if err != nil {
							l.WithError(err).Errorf("Unable to update [%d] buddy list to set [%d] as offline.", targetId, characterId)
						}
					}
					return nil
				})
			}
		}
	}
}

func Accept(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
		t := tenant.MustFromContext(ctx)
		statusEventProducer := producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)
		return func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
			return func(characterId uint32, worldId byte, targetId uint32) error {
				return db.Transaction(func(tx *gorm.DB) error {
					cbl, err := GetByCharacterId(l)(ctx)(tx)(characterId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve buddy list for character [%d] attempting to add buddy.", characterId)
						err = statusEventProducer(errorStatusEventProvider(characterId, worldId, StatusEventErrorUnknownError))
						return err
					}

					if byte(len(cbl.Buddies()))+1 > cbl.Capacity() {
						l.Infof("Buddy list for character [%d] is at capacity.", characterId)
						err = statusEventProducer(errorStatusEventProvider(characterId, worldId, StatusEventErrorListFull))
						return nil
					}

					var found = false
					for _, b := range cbl.Buddies() {
						if b.CharacterId() == targetId {
							found = true
							break
						}
					}
					if found {
						l.Infof("Target [%d] is already on character [%d] buddy list.", targetId, characterId)
						err = statusEventProducer(errorStatusEventProvider(characterId, worldId, StatusEventErrorAlreadyBuddy))
						return nil
					}

					obl, err := GetByCharacterId(l)(ctx)(tx)(targetId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve buddy list for character [%d] attempting to add buddy.", characterId)
						err = statusEventProducer(errorStatusEventProvider(characterId, worldId, StatusEventErrorUnknownError))
						return err
					}
					var ob buddy.Model
					for _, b := range obl.Buddies() {
						if b.CharacterId() == characterId {
							ob = b
						}
					}

					c, err := character.GetById(l)(ctx)(characterId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve character [%d] information.", characterId)
						err = statusEventProducer(errorStatusEventProvider(characterId, worldId, StatusEventErrorUnknownError))
						return err
					}

					oc, err := character.GetById(l)(ctx)(targetId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve character [%d] information.", targetId)
						err = statusEventProducer(errorStatusEventProvider(characterId, worldId, StatusEventErrorCharacterNotFound))
						return err
					}

					err = removeBuddy(db, t.Id(), targetId, characterId)
					if err != nil {
						return err
					}

					err = addBuddy(db, t.Id(), characterId, targetId, oc.Name(), "Default Group", false)
					if err != nil {
						return err
					}

					err = addBuddy(db, t.Id(), targetId, characterId, c.Name(), ob.Group(), false)
					if err != nil {
						return err
					}

					err = statusEventProducer(buddyAddedStatusEventProvider(characterId, worldId, targetId, oc.Name(), -1, "Default Group"))
					if err != nil {
						l.WithError(err).Errorf("Unable to update character [%d] accepting buddy invite from [%d].", characterId, targetId)
					}
					// TODO need to trigger a channel request for target.
					return nil
				})
			}
		}
	}
}

func Delete(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
		t := tenant.MustFromContext(ctx)
		statusEventProducer := producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)
		return func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
			return func(characterId uint32, worldId byte, targetId uint32) error {
				return db.Transaction(func(tx *gorm.DB) error {
					err := removeBuddy(tx, t.Id(), characterId, targetId)
					if err != nil {
						l.WithError(err).Errorf("Unable to remove buddy from buddy list for character [%d].", characterId)
						return err
					}
					var update bool
					update, err = updateBuddyChannel(tx, t.Id(), characterId, targetId, -1)
					if err != nil {
						l.WithError(err).Errorf("Unable to update character [%d] channel to [%d] in [%d] buddy list.", characterId, -1, targetId)
						return err
					}

					err = statusEventProducer(buddyRemovedStatusEventProvider(characterId, worldId, targetId))
					if err != nil {
						l.WithError(err).Errorf("Unable to inform [%d] that their buddy [%d] was removed.", characterId, targetId)
					}

					if update {
						err = statusEventProducer(buddyChannelChangeStatusEventProvider(targetId, worldId, characterId, -1))
						if err != nil {
							l.WithError(err).Errorf("Unable to update [%d] buddy list to set [%d] as offline.", targetId, characterId)
						}
					}
					return nil
				})
			}
		}
	}
}

func UpdateChannel(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, channelId int8) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, channelId int8) error {
		t := tenant.MustFromContext(ctx)
		statusEventProducer := producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)
		return func(db *gorm.DB) func(characterId uint32, worldId byte, channelId int8) error {
			return func(characterId uint32, worldId byte, channelId int8) error {
				return db.Transaction(func(tx *gorm.DB) error {
					bl, err := byCharacterIdEntityProvider(t.Id(), characterId)(tx)()
					if err != nil {
						l.WithError(err).Errorf("Unable to locate buddy list for character [%d].", characterId)
						return err
					}
					for _, b := range bl.Buddies {
						var update bool
						update, err = updateBuddyChannel(tx, t.Id(), characterId, b.CharacterId, channelId)
						if err != nil {
							l.WithError(err).Errorf("Unable to update character [%d] channel to [%d] in [%d] buddy list.", characterId, channelId, b.CharacterId)
							return err
						}

						if update {
							err = statusEventProducer(buddyChannelChangeStatusEventProvider(b.CharacterId, worldId, characterId, channelId))
							if err != nil {
								l.WithError(err).Errorf("Unable to inform character [%d] that [%d] channel has changed.", b.CharacterId, characterId)
							}
						}
					}
					return nil
				})
			}
		}
	}
}
