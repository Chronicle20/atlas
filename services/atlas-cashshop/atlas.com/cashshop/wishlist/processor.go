package wishlist

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]
	GetByCharacterId(characterId uint32) ([]Model, error)
	Add(characterId uint32, serialNumber uint32) (Model, error)
	Delete(characterId uint32, itemId uuid.UUID) error
	DeleteAll(characterId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
	return p
}

func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	return model.SliceMap(Make)(byCharacterIdEntityProvider(p.t.Id(), characterId)(p.db))(model.ParallelMap())
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *ProcessorImpl) Add(characterId uint32, serialNumber uint32) (Model, error) {
	p.l.Debugf("Character [%d] adding [%d] to their wishlist.", characterId, serialNumber)
	return createEntity(p.db, p.t, characterId, serialNumber)
}

func (p *ProcessorImpl) Delete(characterId uint32, itemId uuid.UUID) error {
	p.l.Debugf("Deleting wish list item [%s] for character [%d].", itemId, characterId)
	return deleteEntity(p.ctx)(p.db, p.t.Id(), characterId, itemId)
}

func (p *ProcessorImpl) DeleteAll(characterId uint32) error {
	p.l.Debugf("Deleting wish list for character [%d].", characterId)
	return deleteEntityForCharacter(p.ctx)(p.db, p.t.Id(), characterId)
}
