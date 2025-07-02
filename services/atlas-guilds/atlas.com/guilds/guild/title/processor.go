package title

import (
	"atlas-guilds/database"
	"context"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	CreateDefaults(guildId uint32) ([]Model, error)
	Replace(guildId uint32, titles []string) error
	Clear(guildId uint32) error
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

func (p *ProcessorImpl) CreateDefaults(guildId uint32) ([]Model, error) {
	var results []Model
	var txErr error
	txErr = database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		var err error
		results, err = createDefault(tx, p.t, guildId)
		if err != nil {
			return err
		}
		return nil
	})
	return results, txErr
}

func (p *ProcessorImpl) Replace(guildId uint32, titles []string) error {
	return database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		err := tx.Where("tenant_id = ? AND guild_id = ?", p.t.Id(), guildId).Delete(&Entity{}).Error
		if err != nil {
			return err
		}
		_, err = create(tx, p.t, guildId, titles)
		if err != nil {
			return err
		}
		return nil
	})
}

func (p *ProcessorImpl) Clear(guildId uint32) error {
	err := p.db.Where("tenant_id = ? AND guild_id = ?", p.t.Id(), guildId).Delete(&Entity{}).Error
	if err != nil {
		return err
	}
	return nil
}
