package coupon

import (
	"atlas-cashshop/coupon/reward"
	"fmt"
	"time"

	"github.com/google/uuid"

	couponrules "github.com/Chronicle20/atlas/libs/atlas-constants/coupon"
)

type Builder struct {
	id              uuid.UUID
	batchId         uuid.UUID
	code            string
	description     string
	active          bool
	startsAt        *time.Time
	expiresAt       *time.Time
	maxUses         *uint32
	redemptionCount uint32
	rewards         reward.Rewards
}

// NewBuilder normalizes the code immediately, so a Model can never hold an
// un-normalized code and no caller has to remember to normalize.
func NewBuilder(code string) *Builder {
	return &Builder{code: couponrules.Normalize(code), active: true}
}

func (b *Builder) SetId(id uuid.UUID) *Builder          { b.id = id; return b }
func (b *Builder) SetBatchId(id uuid.UUID) *Builder     { b.batchId = id; return b }
func (b *Builder) SetDescription(d string) *Builder     { b.description = d; return b }
func (b *Builder) SetActive(a bool) *Builder            { b.active = a; return b }
func (b *Builder) SetStartsAt(t *time.Time) *Builder    { b.startsAt = t; return b }
func (b *Builder) SetExpiresAt(t *time.Time) *Builder   { b.expiresAt = t; return b }
func (b *Builder) SetMaxUses(n *uint32) *Builder        { b.maxUses = n; return b }
func (b *Builder) SetRedemptionCount(n uint32) *Builder { b.redemptionCount = n; return b }
func (b *Builder) SetRewards(r reward.Rewards) *Builder { b.rewards = r; return b }

func (b *Builder) Build() (Model, error) {
	if !couponrules.Plausible(b.code) {
		return Model{}, fmt.Errorf("%w: code must be 1-%d characters after normalization", ErrInvalidCoupon, couponrules.MaxCodeLength)
	}
	if err := b.rewards.Validate(); err != nil {
		return Model{}, fmt.Errorf("%w: %s", ErrInvalidCoupon, err)
	}
	if b.startsAt != nil && b.expiresAt != nil && !b.expiresAt.After(*b.startsAt) {
		return Model{}, fmt.Errorf("%w: expiresAt must be after startsAt", ErrInvalidCoupon)
	}
	if b.maxUses != nil && *b.maxUses < b.redemptionCount {
		return Model{}, fmt.Errorf("%w: maxUses (%d) is below the current redemption count (%d)", ErrInvalidCoupon, *b.maxUses, b.redemptionCount)
	}
	return Model{
		id:              b.id,
		batchId:         b.batchId,
		code:            b.code,
		description:     b.description,
		active:          b.active,
		startsAt:        b.startsAt,
		expiresAt:       b.expiresAt,
		maxUses:         b.maxUses,
		redemptionCount: b.redemptionCount,
		rewards:         b.rewards,
	}, nil
}
