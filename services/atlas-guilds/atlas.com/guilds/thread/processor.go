package thread

import (
	"atlas-guilds/database"
	"atlas-guilds/kafka/message"
	thread2 "atlas-guilds/kafka/message/thread"
	"atlas-guilds/kafka/producer"
	"atlas-guilds/thread/reply"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	WithTransaction(tx *gorm.DB) Processor
	AllProvider(guildId uint32) model.Provider[[]Model]
	GetAll(guildId uint32) ([]Model, error)
	ByIdProvider(guildId uint32, threadId uint32) model.Provider[Model]
	GetById(guildId uint32, threadId uint32) (Model, error)

	Create(mb *message.Buffer) func(worldId byte) func(guildId uint32) func(posterId uint32) func(title string) func(message string) func(emoticonId uint32) func(notice bool) (Model, error)
	CreateAndEmit(worldId byte, guildId uint32, posterId uint32, title string, message string, emoticonId uint32, notice bool) (Model, error)
	Update(mb *message.Buffer) func(worldId byte) func(guildId uint32) func(threadId uint32) func(posterId uint32) func(title string) func(message string) func(emoticonId uint32) func(notice bool) (Model, error)
	UpdateAndEmit(worldId byte, guildId uint32, threadId uint32, posterId uint32, title string, message string, emoticonId uint32, notice bool) (Model, error)
	Delete(mb *message.Buffer) func(worldId byte) func(guildId uint32) func(threadId uint32) func(actorId uint32) error
	DeleteAndEmit(worldId byte, guildId uint32, threadId uint32, actorId uint32) error
	Reply(mb *message.Buffer) func(worldId byte) func(guildId uint32) func(threadId uint32) func(posterId uint32) func(message string) (Model, error)
	ReplyAndEmit(worldId byte, guildId uint32, threadId uint32, posterId uint32, message string) (Model, error)
	DeleteReply(mb *message.Buffer) func(worldId byte) func(guildId uint32) func(threadId uint32) func(actorId uint32) func(replyId uint32) (Model, error)
	DeleteReplyAndEmit(worldId byte, guildId uint32, threadId uint32, actorId uint32, replyId uint32) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
}

func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   p.l,
		ctx: p.ctx,
		db:  tx,
		t:   p.t,
	}
}

func (p *ProcessorImpl) AllProvider(guildId uint32) model.Provider[[]Model] {
	return model.SliceMap(Make)(getAll(p.t.Id(), guildId)(p.db))()
}

func (p *ProcessorImpl) GetAll(guildId uint32) ([]Model, error) {
	return p.AllProvider(guildId)()
}

func (p *ProcessorImpl) ByIdProvider(guildId uint32, threadId uint32) model.Provider[Model] {
	return model.Map(Make)(getById(p.t.Id(), guildId, threadId)(p.db))
}

func (p *ProcessorImpl) GetById(guildId uint32, threadId uint32) (Model, error) {
	return p.ByIdProvider(guildId, threadId)()
}

func (p *ProcessorImpl) Create(mb *message.Buffer) func(worldId byte) func(guildId uint32) func(posterId uint32) func(title string) func(message string) func(emoticonId uint32) func(notice bool) (Model, error) {
	return func(worldId byte) func(guildId uint32) func(posterId uint32) func(title string) func(message string) func(emoticonId uint32) func(notice bool) (Model, error) {
		return func(guildId uint32) func(posterId uint32) func(title string) func(message string) func(emoticonId uint32) func(notice bool) (Model, error) {
			return func(posterId uint32) func(title string) func(message string) func(emoticonId uint32) func(notice bool) (Model, error) {
				return func(title string) func(message string) func(emoticonId uint32) func(notice bool) (Model, error) {
					return func(message string) func(emoticonId uint32) func(notice bool) (Model, error) {
						return func(emoticonId uint32) func(notice bool) (Model, error) {
							return func(notice bool) (Model, error) {
								thr, err := create(p.db, p.t.Id(), guildId, posterId, title, message, emoticonId, notice)
								if err != nil {
									p.l.Debugf("Unable to create thread for guild [%d].", guildId)
									return Model{}, err
								}

								err = mb.Put(thread2.EnvStatusEventTopic, statusEventCreatedProvider(worldId, guildId, thr.Id(), posterId))
								if err != nil {
									p.l.Debugf("Unable to buffer thread [%d] created event for guild [%d].", thr.Id(), guildId)
								}

								return thr, nil
							}
						}
					}
				}
			}
		}
	}
}

func (p *ProcessorImpl) CreateAndEmit(worldId byte, guildId uint32, posterId uint32, title string, msg string, emoticonId uint32, notice bool) (Model, error) {
	var m Model
	err := message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(mb *message.Buffer) error {
		var err error
		m, err = p.Create(mb)(worldId)(guildId)(posterId)(title)(msg)(emoticonId)(notice)
		return err
	})
	return m, err
}

func (p *ProcessorImpl) Update(mb *message.Buffer) func(worldId byte) func(guildId uint32) func(threadId uint32) func(posterId uint32) func(title string) func(message string) func(emoticonId uint32) func(notice bool) (Model, error) {
	return func(worldId byte) func(guildId uint32) func(threadId uint32) func(posterId uint32) func(title string) func(message string) func(emoticonId uint32) func(notice bool) (Model, error) {
		return func(guildId uint32) func(threadId uint32) func(posterId uint32) func(title string) func(message string) func(emoticonId uint32) func(notice bool) (Model, error) {
			return func(threadId uint32) func(posterId uint32) func(title string) func(message string) func(emoticonId uint32) func(notice bool) (Model, error) {
				return func(posterId uint32) func(title string) func(message string) func(emoticonId uint32) func(notice bool) (Model, error) {
					return func(title string) func(message string) func(emoticonId uint32) func(notice bool) (Model, error) {
						return func(message string) func(emoticonId uint32) func(notice bool) (Model, error) {
							return func(emoticonId uint32) func(notice bool) (Model, error) {
								return func(notice bool) (Model, error) {
									var thr Model
									txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
										var err error
										thr, err = p.WithTransaction(tx).GetById(guildId, threadId)
										if err != nil {
											p.l.Debugf("Cannot locate guild [%d] thread [%d] being updated.", guildId, threadId)
											return err
										}
										err = update(tx, p.t.Id(), guildId, threadId, posterId, title, message, emoticonId, notice)
										if err != nil {
											p.l.Debugf("Unable to update guild [%d] thread [%d].", guildId, threadId)
											return err
										}
										return nil
									})
									if txErr != nil {
										return Model{}, txErr
									}

									err := mb.Put(thread2.EnvStatusEventTopic, statusEventUpdatedProvider(worldId, guildId, thr.Id(), posterId))
									if err != nil {
										p.l.Debugf("Unable to buffer thread [%d] updated event for guild [%d].", thr.Id(), guildId)
									}

									return thr, nil
								}
							}
						}
					}
				}
			}
		}
	}
}

func (p *ProcessorImpl) UpdateAndEmit(worldId byte, guildId uint32, threadId uint32, posterId uint32, title string, msg string, emoticonId uint32, notice bool) (Model, error) {
	var m Model
	err := message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(mb *message.Buffer) error {
		var err error
		m, err = p.Update(mb)(worldId)(guildId)(threadId)(posterId)(title)(msg)(emoticonId)(notice)
		return err
	})
	return m, err
}

func (p *ProcessorImpl) Delete(mb *message.Buffer) func(worldId byte) func(guildId uint32) func(threadId uint32) func(actorId uint32) error {
	return func(worldId byte) func(guildId uint32) func(threadId uint32) func(actorId uint32) error {
		return func(guildId uint32) func(threadId uint32) func(actorId uint32) error {
			return func(threadId uint32) func(actorId uint32) error {
				return func(actorId uint32) error {
					txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
						thr, err := getById(p.t.Id(), guildId, threadId)(tx)()
						if err != nil {
							p.l.Debugf("Unable to delete guild [%d] thread [%d].", guildId, threadId)
							return err
						}

						for _, r := range thr.Replies {
							err = reply.NewProcessor(p.l, p.ctx, tx).Delete(threadId, r.Id)
							if err != nil {
								p.l.Debugf("Unable to delete guild [%d] thread [%d].", guildId, threadId)
								return err
							}
						}

						err = remove(tx, p.t.Id(), guildId, threadId)
						if err != nil {
							p.l.Debugf("Unable to delete guild [%d] thread [%d].", guildId, threadId)
							return err
						}
						return nil
					})
					if txErr != nil {
						p.l.Debugf("Unable to delete guild [%d] thread [%d].", guildId, threadId)
						return txErr
					}

					err := mb.Put(thread2.EnvStatusEventTopic, statusEventDeletedProvider(worldId, guildId, threadId, actorId))
					if err != nil {
						p.l.Debugf("Unable to buffer thread [%d] deleted event for guild [%d].", threadId, guildId)
					}

					return nil
				}
			}
		}
	}
}

func (p *ProcessorImpl) DeleteAndEmit(worldId byte, guildId uint32, threadId uint32, actorId uint32) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(mb *message.Buffer) error {
		return p.Delete(mb)(worldId)(guildId)(threadId)(actorId)
	})
}

func (p *ProcessorImpl) Reply(mb *message.Buffer) func(worldId byte) func(guildId uint32) func(threadId uint32) func(posterId uint32) func(message string) (Model, error) {
	return func(worldId byte) func(guildId uint32) func(threadId uint32) func(posterId uint32) func(message string) (Model, error) {
		return func(guildId uint32) func(threadId uint32) func(posterId uint32) func(message string) (Model, error) {
			return func(threadId uint32) func(posterId uint32) func(message string) (Model, error) {
				return func(posterId uint32) func(message string) (Model, error) {
					return func(msg string) (Model, error) {
						var thr Model
						var rp reply.Model
						txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
							_, err := p.WithTransaction(tx).GetById(guildId, threadId)
							if err != nil {
								p.l.Debugf("Unable to locate thread [%d] for guild [%d] being replied to.", guildId, threadId)
								return err
							}
							rp, err = reply.NewProcessor(p.l, p.ctx, tx).Add(threadId, posterId, msg)
							if err != nil {
								p.l.Debugf("Unable to add reply to guild [%d] thread [%d].", guildId, threadId)
								return err
							}
							thr, err = p.WithTransaction(tx).GetById(guildId, threadId)
							if err != nil {
								p.l.Debugf("Unable to locate updated thread [%d] for guild [%d].", guildId, threadId)
								return err
							}
							return nil
						})
						if txErr != nil {
							return Model{}, txErr
						}

						err := mb.Put(thread2.EnvStatusEventTopic, statusEventReplyAddedProvider(worldId, guildId, threadId, posterId, rp.Id()))
						if err != nil {
							p.l.Debugf("Unable to buffer thread [%d] replied to event for guild [%d].", thr.Id(), guildId)
						}

						return thr, nil
					}
				}
			}
		}
	}
}

func (p *ProcessorImpl) ReplyAndEmit(worldId byte, guildId uint32, threadId uint32, posterId uint32, msg string) (Model, error) {
	var m Model
	err := message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(mb *message.Buffer) error {
		var err error
		m, err = p.Reply(mb)(worldId)(guildId)(threadId)(posterId)(msg)
		return err
	})
	return m, err
}

func (p *ProcessorImpl) DeleteReply(mb *message.Buffer) func(worldId byte) func(guildId uint32) func(threadId uint32) func(actorId uint32) func(replyId uint32) (Model, error) {
	return func(worldId byte) func(guildId uint32) func(threadId uint32) func(actorId uint32) func(replyId uint32) (Model, error) {
		return func(guildId uint32) func(threadId uint32) func(actorId uint32) func(replyId uint32) (Model, error) {
			return func(threadId uint32) func(actorId uint32) func(replyId uint32) (Model, error) {
				return func(actorId uint32) func(replyId uint32) (Model, error) {
					return func(replyId uint32) (Model, error) {
						var thr Model
						txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
							_, err := p.WithTransaction(tx).GetById(guildId, threadId)
							if err != nil {
								p.l.Debugf("Unable to locate thread [%d] for guild [%d] being replied to.", guildId, threadId)
								return err
							}
							err = reply.NewProcessor(p.l, p.ctx, tx).Delete(threadId, replyId)
							if err != nil {
								p.l.Debugf("Unable to add reply to guild [%d] thread [%d].", guildId, threadId)
								return err
							}
							thr, err = p.WithTransaction(tx).GetById(guildId, threadId)
							if err != nil {
								p.l.Debugf("Unable to locate updated thread [%d] for guild [%d].", guildId, threadId)
								return err
							}
							return nil
						})
						if txErr != nil {
							return Model{}, txErr
						}

						err := mb.Put(thread2.EnvStatusEventTopic, statusEventReplyDeletedProvider(worldId, guildId, threadId, actorId, replyId))
						if err != nil {
							p.l.Debugf("Unable to buffer thread [%d] reply removed event for guild [%d].", thr.Id(), guildId)
						}

						return thr, nil
					}
				}
			}
		}
	}
}

func (p *ProcessorImpl) DeleteReplyAndEmit(worldId byte, guildId uint32, threadId uint32, actorId uint32, replyId uint32) (Model, error) {
	var m Model
	err := message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(mb *message.Buffer) error {
		var err error
		m, err = p.DeleteReply(mb)(worldId)(guildId)(threadId)(actorId)(replyId)
		return err
	})
	return m, err
}
