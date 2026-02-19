package character

import (
	"context"

	database "github.com/Chronicle20/atlas-database"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	ByIdProvider(characterId uint32) model.Provider[Model]
	GetById(characterId uint32) (Model, error)
	SetGuild(characterId uint32, guildId uint32) error
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

func (p *ProcessorImpl) ByIdProvider(characterId uint32) model.Provider[Model] {
	return model.Map(Make)(getById(characterId)(p.db.WithContext(p.ctx)))
}

func (p *ProcessorImpl) GetById(characterId uint32) (Model, error) {
	return p.ByIdProvider(characterId)()
}

func (p *ProcessorImpl) SetGuild(characterId uint32, guildId uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		c, _ := getById(characterId)(tx)()
		if c.GuildId != 0 {
			c.GuildId = guildId
			return tx.Save(&c).Error
		}
		c = Entity{
			TenantId:    p.t.Id(),
			CharacterId: characterId,
			GuildId:     guildId,
		}
		return tx.Save(&c).Error
	})
}
