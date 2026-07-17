package wish

import (
	"context"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// Processor exposes the REST-facing operations over wish-list entries. Kafka
// emission (the MethodAndEmit convention) lands in a later phase; for now these
// are REST-only reads/writes.
type Processor interface {
	GetById(id string) (Model, error)
	GetBySerial(worldId world.Id, sn uint32) (Model, error)
	Create(m Model) (Model, error)
	GetByCharacter(characterId uint32) ([]Model, error)
	GetByCharacterAndType(characterId uint32, wishType string) ([]Model, error)
	// GetWantedByWorld returns every want-ad in a world, across all characters —
	// the cross-character Wanted tab.
	GetWantedByWorld(worldId world.Id) ([]Model, error)
	// ByCharacterPagedProvider returns one page of a character's wishlist (the
	// REST list handler's unfiltered branch, task-117).
	ByCharacterPagedProvider(characterId uint32, page model.Page) model.Provider[model.Paged[Model]]
	// ByCharacterAndTypePagedProvider returns one page of a character's wishes of
	// one kind (the REST list handler's ?type= branch, task-117).
	ByCharacterAndTypePagedProvider(characterId uint32, wishType string, page model.Page) model.Provider[model.Paged[Model]]
	// WantedByWorldPagedProvider returns one page of every want-ad in a world,
	// across all characters (task-117).
	WantedByWorldPagedProvider(worldId world.Id, page model.Page) model.Provider[model.Paged[Model]]
	Delete(id string) (bool, error)
	// DeleteBySerial resolves a wish entry by its ITC serial and deletes it, returning
	// true iff a row was removed. Consumes a fulfilled want-ad on an offer purchase.
	DeleteBySerial(worldId world.Id, sn uint32) (bool, error)
	// RegisterWish creates a wish-list entry in one local DB transaction, deriving
	// the want-ad base price + fixed-sale expiry for a "wanted" entry. It is the
	// row-create business logic behind the RegisterWish command.
	RegisterWish(req RegisterWishRequest) error
	// RemoveWish deletes a wish-list entry by id in one local DB transaction,
	// returning the owning characterId (0 if the row was already gone) so the
	// consumer can echo it onto the WISH_REMOVED event.
	RemoveWish(id string) (uint32, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, db: db}
}

func (p *ProcessorImpl) GetById(id string) (Model, error) {
	return GetById(id)(p.db.WithContext(p.ctx))()
}

// GetBySerial resolves a wish entry by its per-(tenant, world) ITC serial.
func (p *ProcessorImpl) GetBySerial(worldId world.Id, sn uint32) (Model, error) {
	return GetBySerial(worldId, sn)(p.db.WithContext(p.ctx))()
}

// Create persists a new wish entry and returns the stored Model (with its
// assigned surrogate id).
func (p *ProcessorImpl) Create(m Model) (Model, error) {
	return CreateWish(p.db.WithContext(p.ctx), m)
}

// GetByCharacter returns the wish entries for a character. The signature mirrors
// the getByCharacter provider exactly.
func (p *ProcessorImpl) GetByCharacter(characterId uint32) ([]Model, error) {
	return model.SliceMap(modelFromEntity)(getByCharacter(characterId)(p.db.WithContext(p.ctx)))()()
}

// GetByCharacterAndType returns a character's wishes of one kind (cart/wanted),
// so the Cart and Wanted views stay disjoint.
func (p *ProcessorImpl) GetByCharacterAndType(characterId uint32, wishType string) ([]Model, error) {
	return model.SliceMap(modelFromEntity)(getByCharacterAndType(characterId, wishType)(p.db.WithContext(p.ctx)))()()
}

// GetWantedByWorld returns every want-ad in a world, across all characters. The
// signature mirrors the getWantedByWorld provider exactly.
func (p *ProcessorImpl) GetWantedByWorld(worldId world.Id) ([]Model, error) {
	return model.SliceMap(modelFromEntity)(getWantedByWorld(worldId)(p.db.WithContext(p.ctx)))()()
}

// ByCharacterPagedProvider returns one page of a character's wishlist.
func (p *ProcessorImpl) ByCharacterPagedProvider(characterId uint32, page model.Page) model.Provider[model.Paged[Model]] {
	return model.MapPaged(modelFromEntity)(getByCharacterPaged(characterId, page)(p.db.WithContext(p.ctx)))(model.ParallelMap())
}

// ByCharacterAndTypePagedProvider returns one page of a character's wishes of one kind.
func (p *ProcessorImpl) ByCharacterAndTypePagedProvider(characterId uint32, wishType string, page model.Page) model.Provider[model.Paged[Model]] {
	return model.MapPaged(modelFromEntity)(getByCharacterAndTypePaged(characterId, wishType, page)(p.db.WithContext(p.ctx)))(model.ParallelMap())
}

// WantedByWorldPagedProvider returns one page of every want-ad in a world, across all characters.
func (p *ProcessorImpl) WantedByWorldPagedProvider(worldId world.Id, page model.Page) model.Provider[model.Paged[Model]] {
	return model.MapPaged(modelFromEntity)(getWantedByWorldPaged(worldId, page)(p.db.WithContext(p.ctx)))(model.ParallelMap())
}

// Delete hard-deletes the wish entry by id, returning true iff exactly one row
// was deleted.
func (p *ProcessorImpl) DeleteBySerial(worldId world.Id, sn uint32) (bool, error) {
	m, err := p.GetBySerial(worldId, sn)
	if err != nil {
		return false, err
	}
	return p.Delete(m.Id().String())
}

func (p *ProcessorImpl) Delete(id string) (bool, error) {
	affected, err := DeleteWish(p.db.WithContext(p.ctx), id)
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}
