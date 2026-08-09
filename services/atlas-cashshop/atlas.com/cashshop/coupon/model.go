package coupon

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	couponrules "github.com/Chronicle20/atlas/libs/atlas-constants/coupon"
)

// The seven client-facing outcome keys. Each is a key in the CashShopOperation
// writer's `errors` table (see docs/packets/dispatchers/cash_shop_operation.yaml),
// EXCEPT ErrorKeyUnknown, which is the jump table's default case on every
// version and is therefore deliberately NOT configured — see Task 23.
const (
	ErrorKeyInvalidCode   = "INVALID_COUPON_CODE"
	ErrorKeyNotRegistered = "COUPON_NOT_REGISTERED"
	ErrorKeyExpired       = "COUPON_EXPIRED"
	ErrorKeyAlreadyUsed   = "COUPON_ALREADY_USED"
	ErrorKeyUsageLimit    = "COUPON_USAGE_LIMIT"
	ErrorKeyInventoryFull = "INVENTORY_FULL"
	ErrorKeyUnknown       = "UNKNOWN_ERROR"
)

// RedemptionError carries the client-facing key, so the mapping from outcome
// to wire key lives once — at the ladder — and no transport layer re-derives it.
type RedemptionError struct {
	key    string
	detail string
}

func NewRedemptionError(key string, detail string) *RedemptionError {
	return &RedemptionError{key: key, detail: detail}
}

func (e *RedemptionError) Key() string { return e.key }

func (e *RedemptionError) Error() string {
	return fmt.Sprintf("coupon redemption rejected [%s]: %s", e.key, e.detail)
}

type Model struct {
	id              uuid.UUID
	batchId         uuid.UUID
	code            string
	description     string
	active          bool
	startsAt        *time.Time
	expiresAt       *time.Time
	maxUses         *uint32
	redemptionCount uint32
	rewards         Rewards
	createdAt       time.Time
	updatedAt       time.Time
}

func (m Model) Id() uuid.UUID           { return m.id }
func (m Model) BatchId() uuid.UUID      { return m.batchId }
func (m Model) Code() string            { return m.code }
func (m Model) Description() string     { return m.description }
func (m Model) Active() bool            { return m.active }
func (m Model) StartsAt() *time.Time    { return m.startsAt }
func (m Model) ExpiresAt() *time.Time   { return m.expiresAt }
func (m Model) MaxUses() *uint32        { return m.maxUses }
func (m Model) RedemptionCount() uint32 { return m.redemptionCount }
func (m Model) Rewards() Rewards        { return m.rewards }
func (m Model) CreatedAt() time.Time    { return m.createdAt }
func (m Model) UpdatedAt() time.Time    { return m.updatedAt }

// RedeemableAt runs steps 2-4 and 6 of the FR-5.4 validation ladder — the
// checks answerable from this row alone. FIRST FAILURE WINS, in this exact
// order, so the error a player sees is deterministic. Steps 1 (existence),
// 5 (per-account prior redemption) and 7 (locker capacity) need a query and
// live in the processor.
//
// The usage-limit check here is a FRIENDLY-ERROR FAST PATH only. The real
// enforcement is the conditional UPDATE in administrator.go — a read-then-write
// on redemptionCount is a race and is banned.
func (m Model) RedeemableAt(now time.Time) error {
	if !m.active {
		return NewRedemptionError(ErrorKeyNotRegistered, "coupon is inactive")
	}
	if m.startsAt != nil && now.Before(*m.startsAt) {
		return NewRedemptionError(ErrorKeyNotRegistered, "coupon has not started")
	}
	if m.expiresAt != nil && now.After(*m.expiresAt) {
		return NewRedemptionError(ErrorKeyExpired, "coupon has expired")
	}
	if m.maxUses != nil && m.redemptionCount >= *m.maxUses {
		return NewRedemptionError(ErrorKeyUsageLimit, "coupon has no uses remaining")
	}
	return nil
}

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
	rewards         Rewards
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
func (b *Builder) SetRewards(r Rewards) *Builder        { b.rewards = r; return b }

var ErrInvalidCoupon = errors.New("invalid coupon")

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
