package list

import (
	"atlas-buddies/buddy"
	"atlas-buddies/character"
	"atlas-buddies/invite"
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
		return func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32, group string) error {
			return func(characterId uint32, worldId byte, targetId uint32, group string) error {
				return db.Transaction(func(tx *gorm.DB) error {
					cbl, err := GetByCharacterId(l)(ctx)(tx)(characterId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve buddy list for character [%d] attempting to add buddy.", characterId)
						// TODO send error to requester
						return err
					}

					if byte(len(cbl.Buddies()))+1 > cbl.Capacity() {
						l.Infof("Buddy list for character [%d] is at capacity.", characterId)
						// TODO send error to requester
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
						// TODO send error to requester
						return nil
					}

					tc, err := character.GetById(l)(ctx)(targetId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve character [%d] information.", targetId)
					}

					// soft allocate buddy for character
					err = addPendingBuddy(tx, t.Id(), characterId, targetId, tc.Name(), group)
					if err != nil {
						l.WithError(err).Errorf("Unable to add buddy to buddy list for character [%d].", characterId)
						return err
					}
					err = invite.Create(l)(ctx)(characterId, worldId, targetId)
					if err != nil {

					}
					// TODO respond to requester
					return nil
				})
			}
		}
	}
}

func RequestDelete(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
			return func(characterId uint32, worldId byte, targetId uint32) error {
				return db.Transaction(func(tx *gorm.DB) error {
					cbl, err := GetByCharacterId(l)(ctx)(tx)(characterId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve buddy list for character [%d] attempting to add buddy.", characterId)
						// TODO send error to requester
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
							return err
						}
						return nil
					}

					err = removeBuddy(tx, t.Id(), characterId, targetId)
					if err != nil {
						l.WithError(err).Errorf("Unable to remove buddy from buddy list for character [%d].", characterId)
						return err
					}
					// TODO respond to requester
					return nil
				})
			}
		}
	}
}

func Accept(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
			return func(characterId uint32, worldId byte, targetId uint32) error {
				return db.Transaction(func(tx *gorm.DB) error {
					cbl, err := GetByCharacterId(l)(ctx)(tx)(characterId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve buddy list for character [%d] attempting to add buddy.", characterId)
						// TODO send error to requester
						return err
					}

					if byte(len(cbl.Buddies()))+1 > cbl.Capacity() {
						l.Infof("Buddy list for character [%d] is at capacity.", characterId)
						// TODO send error to requester
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
						// TODO send error to requester
						return nil
					}

					obl, err := GetByCharacterId(l)(ctx)(tx)(targetId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve buddy list for character [%d] attempting to add buddy.", characterId)
						// TODO send error to requester
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
						return err
					}

					oc, err := character.GetById(l)(ctx)(targetId)
					if err != nil {
						l.WithError(err).Errorf("Unable to retrieve character [%d] information.", targetId)
						return err
					}

					err = removeBuddy(db, t.Id(), targetId, characterId)
					if err != nil {
						return err
					}

					err = addBuddy(db, t.Id(), characterId, targetId, oc.Name(), ob.Group(), false)
					if err != nil {
						return err
					}

					err = addBuddy(db, t.Id(), targetId, characterId, c.Name(), ob.Group(), false)
					if err != nil {
						return err
					}
					return nil
				})
			}
		}
	}
}

func Delete(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(characterId uint32, worldId byte, targetId uint32) error {
			return func(characterId uint32, worldId byte, targetId uint32) error {
				err := removeBuddy(db, t.Id(), characterId, targetId)
				if err != nil {
					l.WithError(err).Errorf("Unable to remove buddy from buddy list for character [%d].", characterId)
					return err
				}
				// TODO update buddies entry to have a -1 channel
				// TODO respond to requester
				return nil
			}
		}
	}
}
