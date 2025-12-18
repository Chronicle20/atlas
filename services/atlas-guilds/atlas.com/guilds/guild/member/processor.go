package member

import (
	"atlas-guilds/database"
	"atlas-guilds/guild/character"
	"context"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	AddMember(guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte) (Model, error)
	RemoveMember(guildId uint32, characterId uint32) error
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

func (p *ProcessorImpl) AddMember(guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte) (Model, error) {
	var m Model
	var txErr error
	txErr = database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		var err error
		m, err = create(tx, p.t, guildId, characterId, name, jobId, level, title)
		if err != nil {
			return err
		}

		err = character.NewProcessor(p.l, p.ctx, tx).SetGuild(characterId, guildId)
		if err != nil {
			return err
		}

		return nil
	})
	return m, txErr
}

func (p *ProcessorImpl) RemoveMember(guildId uint32, characterId uint32) error {
	return database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		err := tx.Where("tenant_id = ? AND guild_id = ? AND character_id = ?", p.t.Id(), guildId, characterId).Delete(&Entity{}).Error
		if err != nil {
			return err
		}

		err = character.NewProcessor(p.l, p.ctx, tx).SetGuild(characterId, 0)
		if err != nil {
			return err
		}
		return nil
	})
}
