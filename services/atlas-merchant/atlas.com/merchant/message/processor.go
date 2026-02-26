package message

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *Processor {
	return &Processor{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
}

// SendMessage persists a chat message for a shop.
func (p *Processor) SendMessage(shopId uuid.UUID, characterId uint32, content string) error {
	_, err := create(p.t.Id(), shopId, characterId, content)(p.db.WithContext(p.ctx))()
	return err
}

// GetMessages retrieves all messages for a shop.
func (p *Processor) GetMessages(shopId uuid.UUID) ([]Model, error) {
	return model.SliceMap(Make)(getByShopId(shopId)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}
