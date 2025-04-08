package list

import (
	"atlas-buddies/buddy"
	"atlas-buddies/character"
	"atlas-buddies/invite"
	list2 "atlas-buddies/kafka/message/list"
	"atlas-buddies/kafka/producer"
	list3 "atlas-buddies/kafka/producer/list"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
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

func Delete(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, worldId byte) error {
			return func(characterId uint32, worldId byte) error {
				var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})

				txErr := db.Transaction(func(tx *gorm.DB) error {
					bl, err := GetByCharacterId(l)(ctx)(db)(characterId)
					if err != nil {
						return err
					}

					// Remove deleted character for all of their buddies.
					for _, b := range bl.Buddies() {
						err = removeBuddy(tx, t.Id(), b.CharacterId(), characterId)
						if err != nil {
							l.WithError(err).Errorf("Unable to remove buddy from buddy list for character [%d].", b.CharacterId())
							return err
						}

						events = model.MergeSliceProvider(events, list3.BuddyRemovedStatusEventProvider(b.CharacterId(), worldId, characterId))
						if err != nil {
							l.WithError(err).Errorf("Unable to inform [%d] that their buddy [%d] was removed.", b.CharacterId(), characterId)
						}
					}
					return deleteEntityWithBuddies(tx, t.Id(), characterId)
				})
				if txErr != nil {
					return txErr
				}
				msgErr := producer.ProviderImpl(l)(ctx)(list2.EnvStatusEventTopic)(events)
				if msgErr != nil {
					return msgErr
				}
				return nil
			}
		}
	}
}

func RequestAddBuddy(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32, group string) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32, group string) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32, group string) error {
			return func(characterId uint32, worldId byte, targetId uint32, group string) error {
				var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})
				txErr := db.Transaction(func(tx *gorm.DB) error {
					tc, err := character.GetById(l)(ctx)(targetId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve character [%d] information.", targetId)
						events = model.MergeSliceProvider(events, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorCharacterNotFound))
						return err
					}

					if tc.GM() > 0 {
						l.Infof("Character [%d] attempting to buddy a GM.", characterId)
						events = model.MergeSliceProvider(events, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorCannotBuddyGm))
						return nil
					}

					cbl, err := GetByCharacterId(l)(ctx)(tx)(characterId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve buddy list for character [%d] attempting to add buddy.", characterId)
						events = model.MergeSliceProvider(events, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
						return err
					}

					if byte(len(cbl.Buddies()))+1 > cbl.Capacity() {
						l.Infof("Buddy list for character [%d] is at capacity.", characterId)
						events = model.MergeSliceProvider(events, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorListFull))
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
						events = model.MergeSliceProvider(events, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorAlreadyBuddy))
						return err
					}

					obl, err := GetByCharacterId(l)(ctx)(tx)(targetId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve buddy list for character [%d] being added as buddy.", targetId)
						events = model.MergeSliceProvider(events, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
						return err
					}

					if byte(len(obl.Buddies()))+1 > obl.Capacity() {
						l.Infof("Buddy list for character [%d] is at capacity.", targetId)
						events = model.MergeSliceProvider(events, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorOtherListFull))
						return err
					}

					var mbe *buddy.Model
					for _, b := range obl.Buddies() {
						if b.CharacterId() == characterId {
							mbe = &b
							break
						}
					}
					if mbe != nil {
						l.Infof("Character [%d] is already on target characters [%d] buddy list.", characterId, targetId)
						err = addBuddy(tx, t.Id(), characterId, targetId, tc.Name(), group, false)
						if err != nil {
							l.WithError(err).Errorf("Unable to add buddy to buddy list for character [%d].", characterId)
							events = model.MergeSliceProvider(events, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
							return err
						}

						events = model.MergeSliceProvider(events, list3.BuddyAddedStatusEventProvider(characterId, worldId, targetId, tc.Name(), -1, group))
						events = model.MergeSliceProvider(events, list3.BuddyAddedStatusEventProvider(targetId, worldId, characterId, mbe.Name(), mbe.ChannelId(), mbe.Group()))
						// TODO need to trigger a channel request for target.
						return nil
					}

					// soft allocate buddy for character
					err = addPendingBuddy(tx, t.Id(), characterId, targetId, tc.Name(), group)
					if err != nil {
						l.WithError(err).Errorf("Unable to add buddy to buddy list for character [%d].", characterId)
						events = model.MergeSliceProvider(events, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
						return err
					}
					err = invite.Create(l)(ctx)(characterId, worldId, targetId)
					if err != nil {
						events = model.MergeSliceProvider(events, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
						return err
					}
					events = model.MergeSliceProvider(events, list3.BuddyAddedStatusEventProvider(characterId, worldId, targetId, tc.Name(), -1, group))
					return nil
				})
				msgErr := producer.ProviderImpl(l)(ctx)(list2.EnvStatusEventTopic)(events)
				if msgErr != nil {
					return msgErr
				}
				return txErr
			}
		}
	}
}

func RequestDeleteBuddy(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
			return func(characterId uint32, worldId byte, targetId uint32) error {
				var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})
				txErr := db.Transaction(func(tx *gorm.DB) error {
					cbl, err := GetByCharacterId(l)(ctx)(tx)(characterId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve buddy list for character [%d] attempting to add buddy.", characterId)
						events = model.MergeSliceProvider(events, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
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
							events = model.MergeSliceProvider(events, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
							return err
						}
						return nil
					}

					err = removeBuddy(tx, t.Id(), characterId, targetId)
					if err != nil {
						l.WithError(err).Errorf("Unable to remove buddy from buddy list for character [%d].", characterId)
						events = model.MergeSliceProvider(events, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
						return err
					}

					var update bool
					update, err = updateBuddyChannel(tx, t.Id(), characterId, targetId, -1)
					if err != nil {
						l.WithError(err).Errorf("Unable to update character [%d] channel to [%d] in [%d] buddy list.", characterId, -1, targetId)
						events = model.MergeSliceProvider(events, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
						return err
					}

					events = model.MergeSliceProvider(events, list3.BuddyRemovedStatusEventProvider(characterId, worldId, targetId))

					if update {
						events = model.MergeSliceProvider(events, list3.BuddyChannelChangeStatusEventProvider(targetId, worldId, characterId, -1))
						if err != nil {
							l.WithError(err).Errorf("Unable to update [%d] buddy list to set [%d] as offline.", targetId, characterId)
						}
					}
					return nil
				})
				msgErr := producer.ProviderImpl(l)(ctx)(list2.EnvStatusEventTopic)(events)
				if msgErr != nil {
					return msgErr
				}
				return txErr
			}
		}
	}
}

func AcceptInvite(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
			return func(characterId uint32, worldId byte, targetId uint32) error {
				var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})
				txErr := db.Transaction(func(tx *gorm.DB) error {
					cbl, err := GetByCharacterId(l)(ctx)(tx)(characterId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve buddy list for character [%d] attempting to add buddy.", characterId)
						events = model.MergeSliceProvider(events, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
						return err
					}

					if byte(len(cbl.Buddies()))+1 > cbl.Capacity() {
						l.Infof("Buddy list for character [%d] is at capacity.", characterId)
						events = model.MergeSliceProvider(events, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorListFull))
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
						events = model.MergeSliceProvider(events, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorAlreadyBuddy))
						return nil
					}

					obl, err := GetByCharacterId(l)(ctx)(tx)(targetId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve buddy list for character [%d] attempting to add buddy.", characterId)
						events = model.MergeSliceProvider(events, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
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
						events = model.MergeSliceProvider(events, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorUnknownError))
						return err
					}

					oc, err := character.GetById(l)(ctx)(targetId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve character [%d] information.", targetId)
						events = model.MergeSliceProvider(events, list3.ErrorStatusEventProvider(characterId, worldId, list2.StatusEventErrorCharacterNotFound))
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

					events = model.MergeSliceProvider(events, list3.BuddyAddedStatusEventProvider(characterId, worldId, targetId, oc.Name(), -1, "Default Group"))
					if err != nil {
						l.WithError(err).Errorf("Unable to update character [%d] accepting buddy invite from [%d].", characterId, targetId)
					}
					// TODO need to trigger a channel request for target.
					return nil
				})
				msgErr := producer.ProviderImpl(l)(ctx)(list2.EnvStatusEventTopic)(events)
				if msgErr != nil {
					return msgErr
				}
				return txErr
			}
		}
	}
}

func DeleteBuddy(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
			return func(characterId uint32, worldId byte, targetId uint32) error {
				var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})
				txErr := db.Transaction(func(tx *gorm.DB) error {
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

					events = model.MergeSliceProvider(events, list3.BuddyRemovedStatusEventProvider(characterId, worldId, targetId))
					if err != nil {
						l.WithError(err).Errorf("Unable to inform [%d] that their buddy [%d] was removed.", characterId, targetId)
					}

					if update {
						events = model.MergeSliceProvider(events, list3.BuddyChannelChangeStatusEventProvider(targetId, worldId, characterId, -1))
						if err != nil {
							l.WithError(err).Errorf("Unable to update [%d] buddy list to set [%d] as offline.", targetId, characterId)
						}
					}
					return nil
				})
				if txErr != nil {
					return txErr
				}
				msgErr := producer.ProviderImpl(l)(ctx)(list2.EnvStatusEventTopic)(events)
				if msgErr != nil {
					return msgErr
				}
				return nil
			}
		}
	}
}

func UpdateBuddyChannel(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, channelId int8) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, channelId int8) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, worldId byte, channelId int8) error {
			return func(characterId uint32, worldId byte, channelId int8) error {
				var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})
				txErr := db.Transaction(func(tx *gorm.DB) error {
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
							events = model.MergeSliceProvider(events, list3.BuddyChannelChangeStatusEventProvider(b.CharacterId, worldId, characterId, channelId))
							if err != nil {
								l.WithError(err).Errorf("Unable to inform character [%d] that [%d] channel has changed.", b.CharacterId, characterId)
							}
						}
					}
					return nil
				})
				msgErr := producer.ProviderImpl(l)(ctx)(list2.EnvStatusEventTopic)(events)
				if msgErr != nil {
					return msgErr
				}
				return txErr
			}
		}
	}
}

func UpdateBuddyShopStatus(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, inShop bool) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, inShop bool) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, worldId byte, inShop bool) error {
			return func(characterId uint32, worldId byte, inShop bool) error {
				var events = model.FixedProvider[[]kafka.Message]([]kafka.Message{})
				txErr := db.Transaction(func(tx *gorm.DB) error {
					bl, err := byCharacterIdEntityProvider(t.Id(), characterId)(tx)()
					if err != nil {
						l.WithError(err).Errorf("Unable to locate buddy list for character [%d].", characterId)
						return err
					}
					for _, b := range bl.Buddies {
						var update bool
						update, err = updateBuddyShopStatus(tx, t.Id(), characterId, b.CharacterId, inShop)
						if err != nil {
							l.WithError(err).Errorf("Unable to update character [%d] shop status to [%t] in [%d] buddy list.", characterId, inShop, b.CharacterId)
							return err
						}

						if update {
							tbl, err := byCharacterIdEntityProvider(t.Id(), b.CharacterId)(tx)()
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

							events = model.MergeSliceProvider(events, list3.BuddyUpdatedStatusEventProvider(b.CharacterId, worldId, tbe.CharacterId, tbe.Group, tbe.CharacterName, b.ChannelId, inShop))
							if err != nil {
								l.WithError(err).Errorf("Unable to inform character [%d] that [%d] has updated.", b.CharacterId, characterId)
							}
						}
					}
					return nil
				})
				msgErr := producer.ProviderImpl(l)(ctx)(list2.EnvStatusEventTopic)(events)
				if msgErr != nil {
					return msgErr
				}
				return txErr
			}
		}
	}
}
