package recipe

import (
	"atlas-maker/data/itemmake"
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ErrNotFound is returned by GetById when no recipe in the tenant's catalog
// produces the requested item. Task 24 maps it to recipe_not_found (404).
var ErrNotFound = errors.New("recipe not found")

// ErrNoCrystalMapping is returned by GetByLeftover when no group-0 recipe
// consumes the requested leftover as a material. Task 24 maps it to
// no_crystal_mapping (422).
var ErrNoCrystalMapping = errors.New("no crystal mapping for leftover")

// crystallizationGroup is the item-make archive's group digit reserved for
// leftover-to-crystal recipes (design C-6, OQ-3). byLeftover indexes only
// entries in this group; a recipe in any other group that happens to list
// the same item as a material must never satisfy GetByLeftover.
const crystallizationGroup = uint32(0)

// tenantIndex is one tenant's built recipe indexes. It is immutable once
// stored: rebuilding replaces the pointer under the cache lock rather than
// mutating in place.
type tenantIndex struct {
	all        []Model
	byItemId   map[item.Id]Model
	byLeftover map[item.Id]Model
}

// index is the process-wide per-tenant recipe cache. The recipe set is a
// few thousand immutable rows per tenant; it is built lazily on first use
// and invalidated wholesale by Invalidate (called on the seed/ingestion
// signal), never mutated in place. sync.RWMutex, not sync.Once, because the
// cache must be invalidatable.
type index struct {
	mu   sync.RWMutex
	byId map[uuid.UUID]*tenantIndex
}

var recipeIndex = &index{
	byId: map[uuid.UUID]*tenantIndex{},
}

// Invalidate drops tenantId's built indexes so the next lookup rebuilds
// from the upstream catalog. Intended for the seed/ingestion signal
// consumer and for tests.
func Invalidate(tenantId uuid.UUID) {
	recipeIndex.mu.Lock()
	defer recipeIndex.mu.Unlock()
	delete(recipeIndex.byId, tenantId)
}

func (idx *index) get(tenantId uuid.UUID) (*tenantIndex, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	ti, ok := idx.byId[tenantId]
	return ti, ok
}

func (idx *index) build(tenantId uuid.UUID, ms []itemmake.Model) *tenantIndex {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	// Another goroutine may have built it while we waited for the write
	// lock; keep the winner rather than rebuilding again.
	if ti, ok := idx.byId[tenantId]; ok {
		return ti
	}

	ti := &tenantIndex{
		all:        make([]Model, 0, len(ms)),
		byItemId:   make(map[item.Id]Model, len(ms)),
		byLeftover: make(map[item.Id]Model),
	}
	for _, m := range ms {
		rm := modelFromItemMake(m)
		ti.all = append(ti.all, rm)
		ti.byItemId[rm.Id()] = rm
		if rm.Group() == crystallizationGroup {
			for _, mat := range rm.Materials() {
				ti.byLeftover[mat.ItemId] = rm
			}
		}
	}
	idx.byId[tenantId] = ti
	return ti
}

func modelFromItemMake(m itemmake.Model) Model {
	materials := make([]Material, 0, len(m.Recipe()))
	for _, mat := range m.Recipe() {
		materials = append(materials, Material{ItemId: mat.ItemId(), Count: mat.Count()})
	}
	rewards := make([]Reward, 0, len(m.RandomReward()))
	for _, r := range m.RandomReward() {
		rewards = append(rewards, Reward{ItemId: r.ItemId(), ItemNum: r.ItemNum(), Prob: r.Prob()})
	}
	quests := make([]QuestRequirement, 0, len(m.ReqQuest()))
	for _, q := range m.ReqQuest() {
		quests = append(quests, QuestRequirement{QuestId: q.QuestId(), State: q.State()})
	}

	return Model{
		id:                m.Id(),
		group:             m.Group(),
		reqLevel:          m.ReqLevel(),
		reqSkillLevel:     m.ReqSkillLevel(),
		itemNum:           m.ItemNum(),
		tuc:               m.Tuc(),
		meso:              m.Meso(),
		catalyst:          item.Id(m.Catalyst()),
		reqItem:           item.Id(m.ReqItem()),
		reqEquip:          item.Id(m.ReqEquip()),
		materials:         materials,
		randomRewards:     rewards,
		questRequirements: quests,
	}
}

// Processor exposes the recipe catalog looked up by produced item id and by
// group-0 leftover material.
type Processor interface {
	// GetById returns the recipe that produces itemId, or ErrNotFound.
	GetById(itemId item.Id) (Model, error)
	// GetByLeftover returns the group-0 recipe whose sole material is
	// leftoverItemId, or ErrNoCrystalMapping. Recipes in any other group
	// are never consulted, even if they list leftoverItemId as a material
	// (design C-6).
	GetByLeftover(leftoverItemId item.Id) (Model, error)
	// GetAll returns every recipe in the tenant's catalog.
	GetAll() ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	im  itemmake.Processor
}

// NewProcessor builds a Processor backed by im, the atlas-data item-make
// upstream (Task 19).
func NewProcessor(l logrus.FieldLogger, ctx context.Context, im itemmake.Processor) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, im: im}
}

var _ Processor = (*ProcessorImpl)(nil)

// ensureIndex returns the tenant's built indexes, building them from the
// upstream catalog on first use and reusing the cached build thereafter,
// until Invalidate drops it.
func (p *ProcessorImpl) ensureIndex() (*tenantIndex, error) {
	t := tenant.MustFromContext(p.ctx)
	if ti, ok := recipeIndex.get(t.Id()); ok {
		return ti, nil
	}

	ms, err := p.im.GetAll()
	if err != nil {
		return nil, err
	}
	return recipeIndex.build(t.Id(), ms), nil
}

func (p *ProcessorImpl) GetById(itemId item.Id) (Model, error) {
	ti, err := p.ensureIndex()
	if err != nil {
		return Model{}, err
	}
	m, ok := ti.byItemId[itemId]
	if !ok {
		return Model{}, fmt.Errorf("%w: item id [%d]", ErrNotFound, itemId)
	}
	return m, nil
}

func (p *ProcessorImpl) GetByLeftover(leftoverItemId item.Id) (Model, error) {
	ti, err := p.ensureIndex()
	if err != nil {
		return Model{}, err
	}
	m, ok := ti.byLeftover[leftoverItemId]
	if !ok {
		return Model{}, fmt.Errorf("%w: leftover item id [%d]", ErrNoCrystalMapping, leftoverItemId)
	}
	return m, nil
}

func (p *ProcessorImpl) GetAll() ([]Model, error) {
	ti, err := p.ensureIndex()
	if err != nil {
		return nil, err
	}
	return ti.all, nil
}
