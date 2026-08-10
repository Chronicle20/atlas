package batch

import (
	"atlas-cashshop/coupon"
	"atlas-cashshop/coupon/redemption"
	"atlas-cashshop/coupon/reward"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	couponrules "github.com/Chronicle20/atlas/libs/atlas-constants/coupon"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// GenerateInput is one bulk-generation request. It carries no tenant id: the
// tenant is taken from the request context, never from a body.
type GenerateInput struct {
	Count       uint32
	Prefix      string
	Length      int
	Description string
	StartsAt    *time.Time
	ExpiresAt   *time.Time
	Rewards     reward.Rewards
}

type Processor interface {
	GetById(id uuid.UUID) (Model, error)
	AllProvider(page model.Page) (model.Paged[Model], error)
	Generate(in GenerateInput) (Model, []string, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, db: db, t: tenant.MustFromContext(ctx)}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetById(id uuid.UUID) (Model, error) {
	e, err := byIdEntityProvider(p.t, id)(p.db.WithContext(p.ctx))()
	if err != nil {
		return Model{}, err
	}
	m, err := Make(e)
	if err != nil {
		return Model{}, err
	}
	return p.decorateRedeemed(m)
}

func (p *ProcessorImpl) AllProvider(page model.Page) (model.Paged[Model], error) {
	ep := allPagedEntityProvider(p.t, page)(p.db.WithContext(p.ctx))
	paged, err := model.MapPaged(Make)(ep)(model.ParallelMap())()
	if err != nil {
		return model.Paged[Model]{}, err
	}
	// Decorated per PAGE, not per collection (docs/rest-pagination.md §6): the
	// count query runs once per row on the page an admin is actually looking
	// at, not once per batch in the tenant.
	for i, m := range paged.Items {
		decorated, derr := p.decorateRedeemed(m)
		if derr != nil {
			return model.Paged[Model]{}, derr
		}
		paged.Items[i] = decorated
	}
	return paged, nil
}

func (p *ProcessorImpl) decorateRedeemed(m Model) (Model, error) {
	count, err := redemption.CountByBatchId(p.db.WithContext(p.ctx), p.t, m.Id())
	if err != nil {
		return Model{}, err
	}
	return NewBuilder(m.RequestedCount()).
		SetId(m.Id()).
		SetDescription(m.Description()).
		SetGeneratedCount(m.GeneratedCount()).
		SetRedeemedCount(uint32(count)).
		SetCreatedAt(m.CreatedAt()).
		Build()
}

// maxCollisionRetries bounds the per-code retry loop. Exhausting it aborts the
// WHOLE batch — see generateOne.
const maxCollisionRetries = 10

// Generate creates the batch row and its coupons in ONE transaction.
//
// GeneratedCount is written as the requested count up front and the whole
// transaction rolls back if any code cannot be produced, so a persisted batch
// always has exactly as many coupons as it claims. A batch that silently
// produced 497 of 500 is worse than one that failed.
func (p *ProcessorImpl) Generate(in GenerateInput) (Model, []string, error) {
	if err := validateInput(in); err != nil {
		return Model{}, nil, err
	}
	prefix := couponrules.Normalize(in.Prefix)

	bm, err := NewBuilder(in.Count).
		SetDescription(in.Description).
		SetGeneratedCount(in.Count).
		Build()
	if err != nil {
		return Model{}, nil, err
	}

	var created Model
	codes := make([]string, 0, in.Count)
	err = database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		created, err = createEntity(tx, p.t, bm)
		if err != nil {
			return err
		}
		for i := uint32(0); i < in.Count; i++ {
			cm, cerr := generateOne(tx, p.t, prefix, in.Length, created.Id(), in)
			if cerr != nil {
				return cerr
			}
			codes = append(codes, cm.Code())
		}
		return nil
	})
	if err != nil {
		return Model{}, nil, err
	}

	// A freshly generated batch has no redemptions yet, but read it back
	// through the same path a GET uses so one shape is produced everywhere.
	created, err = p.decorateRedeemed(created)
	if err != nil {
		return Model{}, nil, err
	}
	return created, codes, nil
}

// validateInput rejects a bad generation request BEFORE any row is written, so
// the caller maps it to 422 rather than discovering it at INSERT time.
//
// ASCII ASSUMPTION: the prefix length is measured in BYTES (len) to match
// couponrules.Plausible, while coupons.code is varchar(32) and counts
// CHARACTERS. Bytes >= characters, so this gate can only ever be stricter than
// the column — it never lets an over-wide code through. GenerateCode's
// alphabet is ASCII-only by design, so the suffix half is exact.
func validateInput(in GenerateInput) error {
	if in.Count == 0 {
		return fmt.Errorf("%w: count must be positive", ErrInvalidBatch)
	}
	if in.Length <= 0 {
		return fmt.Errorf("%w: length must be positive", ErrInvalidBatch)
	}
	prefix := couponrules.Normalize(in.Prefix)
	if len(prefix)+in.Length > couponrules.MaxCodeLength {
		return fmt.Errorf("%w: prefix (%d) plus length (%d) exceeds the %d character code limit",
			ErrInvalidBatch, len(prefix), in.Length, couponrules.MaxCodeLength)
	}
	if err := in.Rewards.Validate(); err != nil {
		return err
	}
	if in.StartsAt != nil && in.ExpiresAt != nil && !in.ExpiresAt.After(*in.StartsAt) {
		return fmt.Errorf("%w: expiresAt must be after startsAt", ErrInvalidBatch)
	}
	return nil
}

// generateOne inserts a single generated coupon, RETRYING on a code collision
// rather than skipping it, so GeneratedCount always equals RequestedCount and
// the caller's "500 codes" really is 500. The unique index on
// (tenant_id, code) is the collision detector — there is no pre-check, which
// would be a race.
//
// Every generated coupon gets MaxUses = 1: a bulk batch exists to hand one
// code to one person.
func generateOne(tx *gorm.DB, t tenant.Model, prefix string, length int, batchId uuid.UUID, in GenerateInput) (coupon.Model, error) {
	for attempt := 0; attempt < maxCollisionRetries; attempt++ {
		suffix, err := coupon.GenerateCode(length)
		if err != nil {
			return coupon.Model{}, err
		}
		one := uint32(1)
		m, err := coupon.NewBuilder(prefix + suffix).
			SetBatchId(batchId).
			SetDescription(in.Description).
			SetStartsAt(in.StartsAt).
			SetExpiresAt(in.ExpiresAt).
			SetMaxUses(&one).
			SetRewards(in.Rewards).
			Build()
		if err != nil {
			return coupon.Model{}, err
		}
		created, err := coupon.CreateEntity(tx, t, m)
		if err == nil {
			return created, nil
		}
		if !coupon.IsDuplicateCode(err) {
			return coupon.Model{}, err
		}
	}
	return coupon.Model{}, fmt.Errorf("unable to generate a unique coupon code after %d attempts; increase the length", maxCollisionRetries)
}
