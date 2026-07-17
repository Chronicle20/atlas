package character

import (
	"atlas-login/inventory"
	"atlas-login/ranking"
	"context"
	"errors"
	"regexp"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/degrade"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	IsValidName(name string) (bool, error)
	ByAccountAndWorldProvider(decorators ...model.Decorator[Model]) func(accountId uint32, worldId world.Id) model.Provider[[]Model]
	GetForWorld(decorators ...model.Decorator[Model]) func(accountId uint32, worldId world.Id) ([]Model, error)
	ByNameProvider(decorators ...model.Decorator[Model]) func(name string) model.Provider[[]Model]
	GetByName(decorators ...model.Decorator[Model]) func(name string) ([]Model, error)
	ByIdProvider(decorators ...model.Decorator[Model]) func(id uint32) model.Provider[Model]
	GetById(decorators ...model.Decorator[Model]) func(id uint32) (Model, error)
	InventoryDecorator() model.Decorator[Model]
	DeleteById(characterId uint32) error
}

type ProcessorImpl struct {
	l        logrus.FieldLogger
	ctx      context.Context
	ip       inventory.Processor
	rankings func(ids []uint32) ([]ranking.Model, error)
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	rp := ranking.NewProcessor(l, ctx)
	p := &ProcessorImpl{
		l:        l,
		ctx:      ctx,
		ip:       inventory.NewProcessor(l, ctx),
		rankings: rp.GetByCharacterIds,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) IsValidName(name string) (bool, error) {
	m, err := regexp.MatchString("^[A-Za-z0-9\u3040-\u309F\u30A0-\u30FF\u4E00-\u9FAF]{3,12}$", name)
	if err != nil {
		return false, err
	}
	if !m {
		return false, nil
	}

	cs, err := p.GetByName()(name)
	if len(cs) != 0 || err != nil {
		return false, nil
	}

	//TODO
	//bn, err := blocked_name.IsBlockedName(l, span)(name)
	//if bn {
	//	return false, err
	//}

	return true, nil
}

// ByAccountAndWorldProvider fetches every character an account has in a
// world. atlas-character's GET /characters?accountId=&worldId= is now
// paginated (task-117); the character-select screen needs the complete set
// (a truncated page would silently hide characters from the player), so
// this drains every page rather than fetching just the first.
func (p *ProcessorImpl) ByAccountAndWorldProvider(decorators ...model.Decorator[Model]) func(accountId uint32, worldId world.Id) model.Provider[[]Model] {
	return func(accountId uint32, worldId world.Id) model.Provider[[]Model] {
		mp := requests.DrainProvider[RestModel, Model](p.l, p.ctx)(byAccountAndWorldUrl(accountId, worldId), 250, Extract, model.Filters[Model]())
		return model.SliceMap(model.Decorate(decorators))(mp)(model.ParallelMap())
	}
}

func (p *ProcessorImpl) GetForWorld(decorators ...model.Decorator[Model]) func(accountId uint32, worldId world.Id) ([]Model, error) {
	return func(accountId uint32, worldId world.Id) ([]Model, error) {
		cs, err := p.ByAccountAndWorldProvider(decorators...)(accountId, worldId)()
		if errors.Is(err, requests.ErrNotFound) {
			return make([]Model, 0), nil
		}
		if err != nil {
			return cs, err
		}
		return p.decorateRankings(cs), nil
	}
}

// decorateRankings applies the slice-level rankings decoration: one bulk
// call for the whole character list (FR-8), failing open to zero-valued
// rank fields on any error, timeout, or missing entry so the character
// select screen always renders. This must never turn a successful
// character-list fetch into a failure.
func (p *ProcessorImpl) decorateRankings(cs []Model) []Model {
	if len(cs) == 0 {
		return cs
	}
	ids := make([]uint32, len(cs))
	for i, c := range cs {
		ids[i] = c.Id()
	}
	rs, err := p.rankings(ids)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to fetch character rankings, character select will render without ranks.")
		// This decoration is slice-level (one bulk call for the whole
		// character list), not per-model like InventoryDecorator, so there
		// is no single entity to attribute the failure to; entityId=0
		// signals a list-wide degradation rather than a specific character.
		degrade.Observe(p.l, "login.character.rankings", 0, err)
		return cs
	}
	return MergeRankings(cs, rs)
}

// MergeRankings rebuilds each character with its ranking values merged in;
// characters without a corresponding ranking entry keep zero-valued rank
// fields. Exported for tests.
func MergeRankings(cs []Model, rs []ranking.Model) []Model {
	byId := make(map[uint32]ranking.Model, len(rs))
	for _, r := range rs {
		byId[r.CharacterId()] = r
	}
	out := make([]Model, len(cs))
	for i, c := range cs {
		r, ok := byId[c.Id()]
		if !ok {
			out[i] = c
			continue
		}
		out[i] = c.ToBuilder().
			SetRank(r.Rank()).
			SetRankMove(r.RankMove()).
			SetJobRank(r.JobRank()).
			SetJobRankMove(r.JobRankMove()).
			Build()
	}
	return out
}

func (p *ProcessorImpl) ByNameProvider(decorators ...model.Decorator[Model]) func(name string) model.Provider[[]Model] {
	return func(name string) model.Provider[[]Model] {
		mp := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByName(name), Extract, model.Filters[Model]())
		return model.SliceMap(model.Decorate(decorators))(mp)(model.ParallelMap())
	}
}

func (p *ProcessorImpl) GetByName(decorators ...model.Decorator[Model]) func(name string) ([]Model, error) {
	return func(name string) ([]Model, error) {
		return p.ByNameProvider(decorators...)(name)()
	}
}

func (p *ProcessorImpl) ByIdProvider(decorators ...model.Decorator[Model]) func(id uint32) model.Provider[Model] {
	return func(id uint32) model.Provider[Model] {
		mp := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(id), Extract)
		return model.Map(model.Decorate(decorators))(mp)
	}
}

func (p *ProcessorImpl) GetById(decorators ...model.Decorator[Model]) func(id uint32) (Model, error) {
	return func(id uint32) (Model, error) {
		return p.ByIdProvider(decorators...)(id)()
	}
}

func (p *ProcessorImpl) InventoryDecorator() model.Decorator[Model] {
	return model.ErrDecorator(
		func(m Model) (Model, error) {
			i, err := p.ip.GetByCharacterId(m.Id())
			if err != nil {
				return m, err
			}
			return m.SetInventory(i), nil
		},
		func(m Model, err error) {
			degrade.Observe(p.l, "login.character.inventory", m.Id(), err)
		},
	)
}

func (p *ProcessorImpl) DeleteById(characterId uint32) error {
	return requestDelete(characterId)(p.l, p.ctx)
}
