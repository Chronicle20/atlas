package thread

import (
	"atlas-guilds/kafka/producer"
	"atlas-guilds/thread/reply"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func allProvider(ctx context.Context) func(db *gorm.DB) func(guildId uint32) model.Provider[[]Model] {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) func(guildId uint32) model.Provider[[]Model] {
		return func(guildId uint32) model.Provider[[]Model] {
			return model.SliceMap(Make)(getAll(t.Id(), guildId)(db))()
		}
	}
}

func GetAll(ctx context.Context) func(db *gorm.DB) func(guildId uint32) ([]Model, error) {
	return func(db *gorm.DB) func(guildId uint32) ([]Model, error) {
		return func(guildId uint32) ([]Model, error) {
			return allProvider(ctx)(db)(guildId)()
		}
	}
}

func byIdProvider(ctx context.Context) func(db *gorm.DB) func(guildId uint32, threadId uint32) model.Provider[Model] {
	t := tenant.MustFromContext(ctx)
	return func(db *gorm.DB) func(guildId uint32, threadId uint32) model.Provider[Model] {
		return func(guildId uint32, threadId uint32) model.Provider[Model] {
			return model.Map(Make)(getById(t.Id(), guildId, threadId)(db))
		}
	}
}

func GetById(ctx context.Context) func(db *gorm.DB) func(guildId uint32, threadId uint32) (Model, error) {
	return func(db *gorm.DB) func(guildId uint32, threadId uint32) (Model, error) {
		return func(guildId uint32, threadId uint32) (Model, error) {
			return byIdProvider(ctx)(db)(guildId, threadId)()
		}
	}
}

func Create(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(worldId byte, guildId uint32, posterId uint32, title string, message string, emoticonId uint32, notice bool) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(worldId byte, guildId uint32, posterId uint32, title string, message string, emoticonId uint32, notice bool) (Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(worldId byte, guildId uint32, posterId uint32, title string, message string, emoticonId uint32, notice bool) (Model, error) {
			return func(worldId byte, guildId uint32, posterId uint32, title string, message string, emoticonId uint32, notice bool) (Model, error) {
				thr, err := create(db, t.Id(), guildId, posterId, title, message, emoticonId, notice)
				if err != nil {
					l.Debugf("Unable to create thread for guild [%d].", guildId)
					return Model{}, err
				}
				err = producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventCreatedProvider(worldId, guildId, thr.Id()))
				if err != nil {
					l.Debugf("Unable to report thread [%d] created for guild [%d].", thr.Id(), guildId)
				}
				return thr, nil
			}
		}
	}
}

func Update(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(worldId byte, guildId uint32, threadId uint32, posterId uint32, title string, message string, emoticonId uint32, notice bool) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(worldId byte, guildId uint32, threadId uint32, posterId uint32, title string, message string, emoticonId uint32, notice bool) (Model, error) {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(worldId byte, guildId uint32, threadId uint32, posterId uint32, title string, message string, emoticonId uint32, notice bool) (Model, error) {
			return func(worldId byte, guildId uint32, threadId uint32, posterId uint32, title string, message string, emoticonId uint32, notice bool) (Model, error) {
				var thr Model
				txErr := db.Transaction(func(tx *gorm.DB) error {
					var err error
					thr, err = GetById(ctx)(tx)(guildId, threadId)
					if err != nil {
						l.Debugf("Cannot locate guild [%d] thread [%d] being updated.", guildId, threadId)
						return err
					}
					err = update(tx, t.Id(), guildId, threadId, posterId, title, message, emoticonId, notice)
					if err != nil {
						l.Debugf("Unable to update guild [%d] thread [%d].", guildId, threadId)
						return err
					}
					return nil
				})
				if txErr != nil {
					return Model{}, txErr
				}
				err := producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventUpdatedProvider(worldId, guildId, thr.Id()))
				if err != nil {
					l.Debugf("Unable to report thread [%d] updated for guild [%d].", thr.Id(), guildId)
				}
				return thr, nil
			}
		}
	}
}

func Delete(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(worldId byte, guildId uint32, threadId uint32) error {
	return func(ctx context.Context) func(db *gorm.DB) func(worldId byte, guildId uint32, threadId uint32) error {
		t := tenant.MustFromContext(ctx)
		return func(db *gorm.DB) func(worldId byte, guildId uint32, threadId uint32) error {
			return func(worldId byte, guildId uint32, threadId uint32) error {
				txErr := db.Transaction(func(tx *gorm.DB) error {
					thr, err := getById(t.Id(), guildId, threadId)(tx)()
					if err != nil {
						l.Debugf("Unable to delete guild [%d] thread [%d].", guildId, threadId)
						return err
					}

					for _, r := range thr.Replies {
						err = reply.Delete(l)(ctx)(tx)(threadId, r.Id)
						if err != nil {
							l.Debugf("Unable to delete guild [%d] thread [%d].", guildId, threadId)
							return err
						}
					}

					err = remove(tx, t.Id(), guildId, threadId)
					if err != nil {
						l.Debugf("Unable to delete guild [%d] thread [%d].", guildId, threadId)
						return err
					}
					return nil
				})
				if txErr != nil {
					l.Debugf("Unable to delete guild [%d] thread [%d].", guildId, threadId)
					return txErr
				}
				err := producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventDeletedProvider(worldId, guildId, threadId))
				if err != nil {
					l.Debugf("Unable to report thread [%d] deleted for guild [%d].", threadId, guildId)
				}
				return nil
			}
		}
	}
}

func Reply(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(worldId byte, guildId uint32, threadId uint32, posterId uint32, message string) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(worldId byte, guildId uint32, threadId uint32, posterId uint32, message string) (Model, error) {
		return func(db *gorm.DB) func(worldId byte, guildId uint32, threadId uint32, posterId uint32, message string) (Model, error) {
			return func(worldId byte, guildId uint32, threadId uint32, posterId uint32, message string) (Model, error) {
				var thr Model
				var rp reply.Model
				txErr := db.Transaction(func(tx *gorm.DB) error {
					_, err := GetById(ctx)(tx)(guildId, threadId)
					if err != nil {
						l.Debugf("Unable to locate thread [%d] for guild [%d] being replied to.", guildId, threadId)
						return err
					}
					rp, err = reply.Add(l)(ctx)(tx)(threadId, posterId, message)
					if err != nil {
						l.Debugf("Unable to add reply to guild [%d] thread [%d].", guildId, threadId)
						return err
					}
					thr, err = GetById(ctx)(tx)(guildId, threadId)
					if err != nil {
						l.Debugf("Unable to locate updated thread [%d] for guild [%d].", guildId, threadId)
						return err
					}
					return nil
				})
				if txErr != nil {
					return Model{}, txErr
				}
				err := producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventReplyAddedProvider(worldId, guildId, threadId, rp.Id()))
				if err != nil {
					l.Debugf("Unable to report thread [%d] replied to for guild [%d].", thr.Id(), guildId)
				}
				return thr, nil
			}
		}
	}
}

func DeleteReply(l logrus.FieldLogger) func(ctx context.Context) func(db *gorm.DB) func(worldId byte, guildId uint32, threadId uint32, replyId uint32) (Model, error) {
	return func(ctx context.Context) func(db *gorm.DB) func(worldId byte, guildId uint32, threadId uint32, replyId uint32) (Model, error) {
		return func(db *gorm.DB) func(worldId byte, guildId uint32, threadId uint32, replyId uint32) (Model, error) {
			return func(worldId byte, guildId uint32, threadId uint32, replyId uint32) (Model, error) {
				var thr Model
				txErr := db.Transaction(func(tx *gorm.DB) error {
					_, err := GetById(ctx)(tx)(guildId, threadId)
					if err != nil {
						l.Debugf("Unable to locate thread [%d] for guild [%d] being replied to.", guildId, threadId)
						return err
					}
					err = reply.Delete(l)(ctx)(tx)(threadId, replyId)
					if err != nil {
						l.Debugf("Unable to add reply to guild [%d] thread [%d].", guildId, threadId)
						return err
					}
					thr, err = GetById(ctx)(tx)(guildId, threadId)
					if err != nil {
						l.Debugf("Unable to locate updated thread [%d] for guild [%d].", guildId, threadId)
						return err
					}
					return nil
				})
				if txErr != nil {
					return Model{}, txErr
				}
				err := producer.ProviderImpl(l)(ctx)(EnvStatusEventTopic)(statusEventReplyDeletedProvider(worldId, guildId, threadId, replyId))
				if err != nil {
					l.Debugf("Unable to report thread [%d] reply removed for guild [%d].", thr.Id(), guildId)
				}
				return thr, nil
			}
		}
	}
}
