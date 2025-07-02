package thread

import (
	"atlas-guilds/database"
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
	Create(worldId byte, guildId uint32, posterId uint32, title string, message string, emoticonId uint32, notice bool) (Model, error)
	Update(worldId byte, guildId uint32, threadId uint32, posterId uint32, title string, message string, emoticonId uint32, notice bool) (Model, error)
	Delete(worldId byte, guildId uint32, threadId uint32, actorId uint32) error
	Reply(worldId byte, guildId uint32, threadId uint32, posterId uint32, message string) (Model, error)
	DeleteReply(worldId byte, guildId uint32, threadId uint32, actorId uint32, replyId uint32) (Model, error)
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

func (p *ProcessorImpl) Create(worldId byte, guildId uint32, posterId uint32, title string, message string, emoticonId uint32, notice bool) (Model, error) {
	thr, err := create(p.db, p.t.Id(), guildId, posterId, title, message, emoticonId, notice)
	if err != nil {
		p.l.Debugf("Unable to create thread for guild [%d].", guildId)
		return Model{}, err
	}
	err = producer.ProviderImpl(p.l)(p.ctx)(thread2.EnvStatusEventTopic)(statusEventCreatedProvider(worldId, guildId, thr.Id(), posterId))
	if err != nil {
		p.l.Debugf("Unable to report thread [%d] created for guild [%d].", thr.Id(), guildId)
	}
	return thr, nil
}

func (p *ProcessorImpl) Update(worldId byte, guildId uint32, threadId uint32, posterId uint32, title string, message string, emoticonId uint32, notice bool) (Model, error) {
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
	err := producer.ProviderImpl(p.l)(p.ctx)(thread2.EnvStatusEventTopic)(statusEventUpdatedProvider(worldId, guildId, thr.Id(), posterId))
	if err != nil {
		p.l.Debugf("Unable to report thread [%d] updated for guild [%d].", thr.Id(), guildId)
	}
	return thr, nil
}

func (p *ProcessorImpl) Delete(worldId byte, guildId uint32, threadId uint32, actorId uint32) error {
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
	err := producer.ProviderImpl(p.l)(p.ctx)(thread2.EnvStatusEventTopic)(statusEventDeletedProvider(worldId, guildId, threadId, actorId))
	if err != nil {
		p.l.Debugf("Unable to report thread [%d] deleted for guild [%d].", threadId, guildId)
	}
	return nil
}

func (p *ProcessorImpl) Reply(worldId byte, guildId uint32, threadId uint32, posterId uint32, message string) (Model, error) {
	var thr Model
	var rp reply.Model
	txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		_, err := p.WithTransaction(tx).GetById(guildId, threadId)
		if err != nil {
			p.l.Debugf("Unable to locate thread [%d] for guild [%d] being replied to.", guildId, threadId)
			return err
		}
		rp, err = reply.NewProcessor(p.l, p.ctx, tx).Add(threadId, posterId, message)
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
	err := producer.ProviderImpl(p.l)(p.ctx)(thread2.EnvStatusEventTopic)(statusEventReplyAddedProvider(worldId, guildId, threadId, posterId, rp.Id()))
	if err != nil {
		p.l.Debugf("Unable to report thread [%d] replied to for guild [%d].", thr.Id(), guildId)
	}
	return thr, nil
}

func (p *ProcessorImpl) DeleteReply(worldId byte, guildId uint32, threadId uint32, actorId uint32, replyId uint32) (Model, error) {
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
	err := producer.ProviderImpl(p.l)(p.ctx)(thread2.EnvStatusEventTopic)(statusEventReplyDeletedProvider(worldId, guildId, threadId, actorId, replyId))
	if err != nil {
		p.l.Debugf("Unable to report thread [%d] reply removed for guild [%d].", thr.Id(), guildId)
	}
	return thr, nil
}
