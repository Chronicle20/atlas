package ring

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/character"
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Processor scopes CreatePair/GetByCharacterId/GetById to the calling
// tenant so downstream callers (the ring purchase path, a future effects
// consumer) do not have to thread tenantId through by hand.
type Processor interface {
	CreatePair(db *gorm.DB, ringType Type, a Half, b Half) (uuid.UUID, error)
	GetByCharacterId(characterId uint32) ([]Model, error)
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

	astP := asset.NewProcessor(p.l, p.ctx, p.db)
	enriched := make([]Model, 0, len(halves))
	for _, half := range halves {
		b := half.Builder()

		if a, err := astP.GetById(half.AssetId()); err == nil {
			b.SetCashId(a.CashId())
		}

		siblings, err := GetByPairId(db, p.t.Id(), half.PairId())
		if err == nil {
			for _, sibling := range siblings {
				if sibling.Id() == half.Id() {
					continue
				}
				if sa, err := astP.GetById(sibling.AssetId()); err == nil {
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
			return nil, err
		}
		enriched = append(enriched, m)
	}
	return enriched, nil
}

func (p *ProcessorImpl) GetById(id uuid.UUID) (Model, error) {
	return GetById(p.db.WithContext(p.ctx), p.t.Id(), id)
}
