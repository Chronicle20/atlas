package coupon

import (
	"atlas-cashshop/cashshop/inventory/compartment"
	"atlas-cashshop/character"
	"atlas-cashshop/configuration"
	"atlas-cashshop/coupon/redemption"
	"atlas-cashshop/coupon/reward"
	"atlas-cashshop/kafka/message"
	kafkacashshop "atlas-cashshop/kafka/message/cashshop"
	couponproducer "atlas-cashshop/kafka/producer/coupon"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	couponrules "github.com/Chronicle20/atlas/libs/atlas-constants/coupon"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	RedeemAndEmit(characterId uint32, code string) error
	GetById(id uuid.UUID) (Model, error)
	GetByCode(code string) (Model, error)
	GetAll(f Filters) ([]Model, error)
	Create(m Model) (Model, error)
	Update(id uuid.UUID, m Model) (Model, error)
	Delete(id uuid.UUID) error
}

// granterFactory is the seam through which the reward loop resolves a granter.
// It is a field rather than a direct call to granterFor because the cash-item
// granter carries a REMOTE commodity client, which a unit test has no service
// to answer.
type granterFactory func(l logrus.FieldLogger, ctx context.Context, r reward.Reward) (rewardGranter, error)

type ProcessorImpl struct {
	l    logrus.FieldLogger
	ctx  context.Context
	db   *gorm.DB
	t    tenant.Model
	chaP character.Processor
	gf   granterFactory
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:    l,
		ctx:  ctx,
		db:   db,
		t:    tenant.MustFromContext(ctx),
		chaP: character.NewProcessor(l, ctx),
		gf:   granterFor,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// RedeemAndEmit runs one redemption attempt end to end.
//
// Success rides the OUTBOX (it asserts a committed state change). Every
// failure goes on the DIRECT producer path outside the transaction: an event
// asserting "nothing happened" must not ride an outbox that implies a commit
// (the same distinction Purchase draws at cashshop/processor.go:100-103).
//
// The coupon code is a SECRET. Nothing below logs its value or labels a metric
// with it — only its length is ever recorded.
func (p *ProcessorImpl) RedeemAndEmit(characterId uint32, code string) error {
	code = couponrules.Normalize(code)

	// Resolve the owning ACCOUNT: the packet arrives on a character session,
	// but wallets and the one-time-per-account rule are account-scoped.
	c, err := p.chaP.GetById(p.chaP.InventoryDecorator)(characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to resolve character [%d] for coupon redemption.", characterId)
		return p.reject(characterId, ErrorKeyUnknown)
	}
	accountId := c.AccountId()

	attempts, window := configuration.GetCouponRateLimit(p.l, p.ctx, p.t.Id())
	limiter := NewLimiter(attempts, window)
	warnIfLimiterUnwired(p.l)
	allowed, lerr := limiter.Allowed(p.ctx, p.t, accountId)
	if lerr != nil {
		// Fail open — a Redis outage must not make every coupon un-redeemable.
		p.l.WithError(lerr).Warnf("Coupon rate limiter unavailable for account [%d]; allowing the attempt.", accountId)
	}
	if !allowed {
		p.l.Infof("Coupon attempt from account [%d] character [%d] blocked by the rate limiter.", accountId, characterId)
		rateLimitedTotal.WithLabelValues(p.t.Id().String()).Inc()
		// INVALID_COUPON_CODE, not a distinct key: a "rate limited" reply
		// would tell an attacker they had found a real code. Both counters are
		// incremented so the two series stay comparable.
		return p.rejectCounted(characterId, accountId, limiter, ErrorKeyInvalidCode)
	}

	if !couponrules.Plausible(code) {
		p.l.Infof("Coupon attempt from account [%d] character [%d] rejected before any lookup: code length [%d] is not plausible.",
			accountId, characterId, len(code))
		return p.rejectCounted(characterId, accountId, limiter, ErrorKeyInvalidCode)
	}

	transactionId := uuid.New()

	var rejectKey string
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(mb *message.Buffer) error {
			_, rerr := p.redeem(mb)(tx, redeemRequest{
				characterId:     characterId,
				accountId:       accountId,
				code:            code,
				compartmentType: lockerTypeFor(c.JobId()),
				transactionId:   transactionId,
			})
			if rerr != nil {
				var re *RedemptionError
				if errors.As(rerr, &re) {
					rejectKey = re.Key()
				} else {
					rejectKey = ErrorKeyUnknown
				}
				// Returning the error rolls the transaction back, which undoes
				// the reservation, the redemption row and every grant. There is
				// nothing to compensate.
				return rerr
			}
			return nil
		})
	})

	if rejectKey != "" {
		p.l.Infof("Coupon redemption rejected for account [%d] character [%d] code length [%d] transaction [%s]: %s",
			accountId, characterId, len(code), transactionId, rejectKey)
		return p.rejectCounted(characterId, accountId, limiter, rejectKey)
	}
	if txErr != nil {
		p.l.WithError(txErr).Errorf("Coupon redemption failed for account [%d] character [%d] transaction [%s].", accountId, characterId, transactionId)
		return p.rejectCounted(characterId, accountId, limiter, ErrorKeyUnknown)
	}

	p.l.Infof("Coupon redeemed by account [%d] character [%d] code length [%d] transaction [%s].", accountId, characterId, len(code), transactionId)
	attemptsTotal.WithLabelValues(p.t.Id().String(), outcomeSuccess).Inc()
	// A player who mistyped before getting it right should not be left one
	// failure away from a block.
	if rerr := limiter.Reset(p.ctx, p.t, accountId); rerr != nil {
		p.l.WithError(rerr).Warnf("Unable to clear the coupon rate-limit counter for account [%d].", accountId)
	}
	return nil
}

// rejectCounted records the failed attempt against the limiter, counts the
// outcome, and emits the client-facing failure on the DIRECT producer path.
func (p *ProcessorImpl) rejectCounted(characterId uint32, accountId uint32, limiter Limiter, key string) error {
	if err := limiter.RecordFailure(p.ctx, p.t, accountId); err != nil {
		p.l.WithError(err).Warnf("Unable to record a failed coupon attempt for account [%d]; brute-force braking is degraded.", accountId)
	}
	return p.reject(characterId, key)
}

func (p *ProcessorImpl) reject(characterId uint32, key string) error {
	attemptsTotal.WithLabelValues(p.t.Id().String(), key).Inc()
	if err := producer.ProviderImpl(p.l)(p.ctx)(kafkacashshop.EnvEventTopicStatus)(
		couponproducer.CouponFailedStatusEventProvider(characterId, key)); err != nil {
		p.l.WithError(err).Errorf("Unable to notify character [%d] that a coupon redemption failed.", characterId)
	}
	// The caller asked for a redemption and did not get one, so RedeemAndEmit
	// reports an error even though the player has already been told why.
	return NewRedemptionError(key, "redemption rejected")
}

type redeemRequest struct {
	characterId     uint32
	accountId       uint32
	code            string
	compartmentType compartment.CompartmentType
	transactionId   uuid.UUID
}

// grantedTotals is the aggregate of every granter's contribution.
//
// It is NEVER used as a "did anything happen?" test. wallet.Award routes every
// currency other than 1 (credit) and 2 (maple points) to prepaid, and
// currencyGranter reports neither field for those — so a prepaid-only coupon
// mutates the wallet and yields an all-zero grantedTotals, byte-identical to a
// granter that did nothing. err == nil is the only success signal.
//
// compartmentId is uuid.Nil for a coupon whose bundle contains NO cash items.
// It names the locker the assetIds live in, so with no assets there is no
// locker to name — and resolving one anyway would make a currency-only coupon
// fail for an account whose locker row is missing, which has nothing to do
// with the reward being granted. Nil compartmentId together with an empty
// assetIds is the normal currency-only shape, not a bug; consumers must key
// off assetIds. Pinned by TestRedeemPrepaidOnlyCouponStillSucceeds.
type grantedTotals struct {
	compartmentId uuid.UUID
	assetIds      []uint32
	maplePoints   uint32
	credit        uint32
}

// redeem is the FR-5.4 ladder plus the grants, all inside one transaction.
func (p *ProcessorImpl) redeem(mb *message.Buffer) func(tx *gorm.DB, req redeemRequest) (grantedTotals, error) {
	return func(tx *gorm.DB, req redeemRequest) (grantedTotals, error) {
		var out grantedTotals

		// 1. code exists
		e, err := byCodeEntityProvider(p.t, req.code)(tx)()
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return out, NewRedemptionError(ErrorKeyInvalidCode, "no such code")
			}
			return out, err
		}
		m, err := Make(e)
		if err != nil {
			return out, err
		}

		// 2-4, 6. active / window / uses-remaining fast path
		if err = m.RedeemableAt(time.Now()); err != nil {
			return out, err
		}

		// 5. this account has no prior redemption
		prior, err := redemption.CountByCouponAndAccount(tx, p.t, m.Id(), req.accountId)
		if err != nil {
			return out, err
		}
		if prior > 0 {
			return out, NewRedemptionError(ErrorKeyAlreadyUsed, "account already redeemed this coupon")
		}

		// 7. locker capacity, pre-flight. The cash-item granter re-checks
		// inside the transaction (Q6); this check exists so the ERROR ORDERING
		// is deterministic — a full locker reports INVENTORY_FULL rather than
		// whichever grant happened to run first. Both are load-bearing.
		//
		// A bundle with no cash items skips the lookup entirely, and
		// out.compartmentId stays uuid.Nil. See grantedTotals for why that is
		// the contract rather than an omission.
		need := uint32(m.Rewards().CashItemCount())
		if need > 0 {
			ccm, cerr := compartment.NewProcessor(p.l, p.ctx, tx).WithTransaction(tx).
				GetByAccountIdAndType(req.accountId, req.compartmentType)
			if cerr != nil {
				if errors.Is(cerr, gorm.ErrRecordNotFound) {
					// The account has no locker of this type AT ALL. That is
					// not a full locker and INVENTORY_FULL would misdirect the
					// player: freeing a slot cannot fix a row that does not
					// exist. All three compartments are provisioned together
					// when the account is created
					// (kafka/consumer/account/consumer.go:60 ->
					// inventory.Create -> createDefaultCompartments), so a
					// missing one means provisioning failed or predates the
					// service — an operator problem, reported as the generic
					// notice and logged loudly rather than dressed up as a
					// player-fixable one.
					p.l.WithError(cerr).Errorf("Account [%d] has no cash locker of type [%d]; cash-shop provisioning is incomplete. Coupon redemption cannot proceed.",
						req.accountId, req.compartmentType)
					return out, NewRedemptionError(ErrorKeyUnknown, "account has no cash locker of this type")
				}
				return out, cerr
			}
			if ccm.Capacity() < uint32(len(ccm.Assets()))+need {
				return out, NewRedemptionError(ErrorKeyInventoryFull, "cash locker has no free slot")
			}
			out.compartmentId = ccm.Id()
		}

		// Atomic reservation (FR-5.5). RowsAffected is the verdict.
		reserved, err := reserveUse(tx, p.t, m.Id())
		if err != nil {
			return out, err
		}
		if !reserved {
			return out, NewRedemptionError(ErrorKeyUsageLimit, "coupon has no uses remaining")
		}

		// The redemption row. A unique violation on
		// (tenant_id, coupon_id, account_id) is the same-account race loser,
		// not a redundant check.
		rm, err := redemption.NewBuilder(m.Id(), req.accountId, req.characterId).
			SetTransactionId(req.transactionId).
			SetRewardsGranted(m.Rewards()).
			SetRedeemedAt(time.Now()).
			Build()
		if err != nil {
			return out, err
		}
		if _, err = redemption.Create(tx, p.t, rm); err != nil {
			if redemption.IsUniqueViolation(err) {
				return out, NewRedemptionError(ErrorKeyAlreadyUsed, "account already redeemed this coupon")
			}
			return out, err
		}

		// Grants. A granter that returns a nil error SUCCEEDED, whatever its
		// grantedReward contains — see grantedTotals.
		rc := redemptionContext{accountId: req.accountId, characterId: req.characterId, compartmentId: out.compartmentId}
		for _, r := range m.Rewards() {
			g, gerr := p.gf(p.l, p.ctx, r)
			if gerr != nil {
				return out, gerr
			}
			got, gerr := g.Grant(mb)(tx, rc, r)
			if gerr != nil {
				return out, gerr
			}
			if got.assetId != 0 {
				out.assetIds = append(out.assetIds, got.assetId)
			}
			out.maplePoints += got.maplePoints
			out.credit += got.credit
		}

		// Committed with the state change, delivered by the outbox.
		return out, mb.Put(kafkacashshop.EnvEventTopicStatus,
			couponproducer.CouponRedeemedStatusEventProvider(req.characterId, out.compartmentId, out.assetIds, out.maplePoints, out.credit))
	}
}

// lockerTypeFor mirrors the branch Purchase uses at
// cashshop/processor.go:130-136.
func lockerTypeFor(jobId job.Id) compartment.CompartmentType {
	switch job.GetType(jobId) {
	case job.TypeExplorer:
		return compartment.TypeExplorer
	case job.TypeCygnus:
		return compartment.TypeCygnus
	default:
		return compartment.TypeLegend
	}
}

// --- admin CRUD (Task 23's only caller) --------------------------------------

func (p *ProcessorImpl) GetById(id uuid.UUID) (Model, error) {
	return model.Map(Make)(byIdEntityProvider(p.t, id)(p.db.WithContext(p.ctx)))()
}

// GetByCode normalizes its argument, so an admin lookup matches on exactly the
// same rule a player's submission does.
func (p *ProcessorImpl) GetByCode(code string) (Model, error) {
	return model.Map(Make)(byCodeEntityProvider(p.t, couponrules.Normalize(code))(p.db.WithContext(p.ctx)))()
}

func (p *ProcessorImpl) GetAll(f Filters) ([]Model, error) {
	if f.Code != nil {
		normalized := couponrules.Normalize(*f.Code)
		f.Code = &normalized
	}
	return model.SliceMap(Make)(allEntityProvider(p.t, f)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

func (p *ProcessorImpl) Create(m Model) (Model, error) {
	return CreateEntity(p.db.WithContext(p.ctx), p.t, m)
}

// Update applies m's admin-editable fields to the coupon named by id. id wins
// over m.Id(), so a caller that built the model from a request body without
// echoing the id back still updates the right row.
func (p *ProcessorImpl) Update(id uuid.UUID, m Model) (Model, error) {
	target, err := cloneWithId(m, id)
	if err != nil {
		return Model{}, err
	}
	return updateEntity(p.db.WithContext(p.ctx), p.t, target)
}

// Delete removes a coupon, returning ErrHasRedemptions when it has already
// been redeemed (Task 23 maps that to HTTP 409).
func (p *ProcessorImpl) Delete(id uuid.UUID) error {
	return deleteEntity(p.db.WithContext(p.ctx), p.t, id)
}

func cloneWithId(m Model, id uuid.UUID) (Model, error) {
	return NewBuilder(m.Code()).
		SetId(id).
		SetBatchId(m.BatchId()).
		SetDescription(m.Description()).
		SetActive(m.Active()).
		SetStartsAt(m.StartsAt()).
		SetExpiresAt(m.ExpiresAt()).
		SetMaxUses(m.MaxUses()).
		SetRedemptionCount(m.RedemptionCount()).
		SetRewards(m.Rewards()).
		Build()
}
