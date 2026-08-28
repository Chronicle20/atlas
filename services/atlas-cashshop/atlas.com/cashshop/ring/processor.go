package ring

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/character"
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Processor scopes CreatePair/GetByCharacterId/GetById to the calling
// tenant so downstream callers (the ring purchase path, a future effects
// consumer) do not have to thread tenantId through by hand.
type Processor interface {
	CreatePair(db *gorm.DB, ringType Type, a Half, b Half) (uuid.UUID, error)
	GetByCharacterId(characterId uint32) ([]Model, error)
	GetByCharacterIdPaged(characterId uint32, page model.Page) (model.Paged[Model], error)
	GetById(id uuid.UUID) (Model, error)
}

type ProcessorImpl struct {
	l    logrus.FieldLogger
	ctx  context.Context
	db   *gorm.DB
	t    tenant.Model
	chaP character.Processor
}

// NewProcessor takes a character.Processor so GetByCharacterId can resolve
// PartnerName (design.md §5: the wiring change this read-model enrichment
// requires -- ring.ProcessorImpl otherwise has no character-service
// client). The one caller today, cashshop.PurchaseRingAndEmit, already
// holds a character.Processor of its own.
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, chaP character.Processor) Processor {
	return &ProcessorImpl{
		l:    l,
		ctx:  ctx,
		db:   db,
		t:    tenant.MustFromContext(ctx),
		chaP: chaP,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// CreatePair takes the caller-supplied handle -- it is called inside the
// ring purchase transaction, so it must run on that transaction's tx, not on
// the processor's own db.
func (p *ProcessorImpl) CreatePair(db *gorm.DB, ringType Type, a Half, b Half) (uuid.UUID, error) {
	return CreatePair(db, p.t.Id(), ringType, a, b)
}

// GetByCharacterId returns every ring half the character holds, enriched
// with CashId, PartnerCashId, and PartnerName (design.md §5: computed at
// read time, never stored on Entity). Every resolution -- this half's own
// asset, the sibling half, the partner's name -- fails soft to the zero
// value: a lookup failure here must not turn into an error for a caller
// that only wants the pair rows (PRD FR-5's channel-side fallback is
// downstream of this and depends on that).
func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	db := p.db.WithContext(p.ctx)
	halves, err := GetByCharacterId(db, p.t.Id(), characterId)
	if err != nil {
		return nil, err
	}

	enriched := make([]Model, 0, len(halves))
	for _, half := range halves {
		enriched = append(enriched, p.enrich(half))
	}
	return enriched, nil
}

// GetByCharacterIdPaged is GetByCharacterId's paged counterpart, backing
// GET /rings?filter[characterId]=. Paging happens at the database (via
// byCharacterIdPagedProvider) so only the requested page's rows are
// enriched, not the character's full holding.
func (p *ProcessorImpl) GetByCharacterIdPaged(characterId uint32, page model.Page) (model.Paged[Model], error) {
	db := p.db.WithContext(p.ctx)
	pe, err := byCharacterIdPagedProvider(p.t, characterId, page)(db)()
	if err != nil {
		return model.Paged[Model]{}, err
	}

	items := make([]Model, 0, len(pe.Items))
	for _, e := range pe.Items {
		half, err := Make(e)
		if err != nil {
			return model.Paged[Model]{}, err
		}
		items = append(items, p.enrich(half))
	}
	return model.Paged[Model]{Items: items, Total: pe.Total, Page: pe.Page}, nil
}

func (p *ProcessorImpl) GetById(id uuid.UUID) (Model, error) {
	half, err := GetById(p.db.WithContext(p.ctx), p.t.Id(), id)
	if err != nil {
		return Model{}, err
	}
	return p.enrich(half), nil
}

// enrich resolves half's CashId, PartnerCashId, and PartnerName at read
// time. CashId and PartnerCashId prefer the value persisted on Entity at
// purchase time (Half.CashId, carried through Make); the AssetId lookup is
// only a fallback for rows written before that column existed, where the
// stored value is 0. Once a ring leaves the locker and is equipped, its
// locker AssetId no longer resolves, so falling back unconditionally would
// leave every equipped ring's ids at 0 -- see
// docs/tasks/task-269-ring-pair-behavior/bug-ring-cash-id-resolves-to-zero.md.
// Every resolution -- this half's own asset, the sibling half, the
// partner's name -- fails soft to the zero value: a lookup failure here
// must not turn into an error for a caller that only wants the pair rows
// (PRD FR-5's channel-side fallback is downstream of this and depends on
// that).
func (p *ProcessorImpl) enrich(half Model) Model {
	db := p.db.WithContext(p.ctx)
	astP := asset.NewProcessor(p.l, p.ctx, p.db)
	b := half.Builder()

	if half.CashId() == 0 {
		if a, err := astP.GetById(half.AssetId()); err == nil {
			b.SetCashId(a.CashId())
		}
	}

	siblings, err := GetByPairId(db, p.t.Id(), half.PairId())
	if err == nil {
		for _, sibling := range siblings {
			if sibling.Id() == half.Id() {
				continue
			}
			if sibling.CashId() != 0 {
				b.SetPartnerCashId(sibling.CashId())
			} else if sa, err := astP.GetById(sibling.AssetId()); err == nil {
				b.SetPartnerCashId(sa.CashId())
			}
			break
		}
	}

	if partner, err := p.chaP.GetById()(half.PartnerCharacterId()); err == nil {
		b.SetPartnerName(partner.Name())
	}

	m, err := b.Build()
	if err != nil {
		return half
	}
	return m
}
