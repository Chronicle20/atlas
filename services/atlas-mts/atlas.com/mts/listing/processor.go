package listing

import (
	"context"
	"fmt"
	"time"

	"atlas-mts/configuration"
	"atlas-mts/holding"
	"atlas-mts/saga"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// SagaEmitter abstracts the saga-command emission so the list flow can be
// exercised without a live Kafka broker. The production implementation is the
// saga package's Processor; tests inject a capturing stub.
type SagaEmitter interface {
	Create(s saga.Saga) error
}

// ListRequest carries the seller-supplied parameters for a TransferToMts list
// initiation. The item reference (assetId + sourceInventoryType) identifies the
// inventory slot to remove; the snapshot itself is looked up by the saga
// expansion, not carried here.
type ListRequest struct {
	WorldId             world.Id
	SellerId            uint32
	SellerName          string
	SaleType            SaleType
	SourceInventoryType byte
	AssetId             uint32
	Quantity            uint32
	ListValue           uint32
	BuyNowPrice         *uint32
	DurationHours       int // auction only; hours from now until the auction ends
	Category            string
	SubCategory         string
}

// listSagaBaseTimeout and listSagaPerStepTimeout define the step-count-scaled
// timeout for the list saga. The orchestrator processes saga steps serially over
// Kafka, so the timeout budget must grow with the number of effective steps; a
// flat timeout rolls back legitimate multi-step sagas (see the preset-creation
// timeout bug, bug_preset_creation_saga_flat_timeout).
const (
	listSagaBaseTimeout    = 10 * time.Second
	listSagaPerStepTimeout = 1 * time.Second
)

// Processor exposes the REST-facing CRUD and state-transition operations over
// marketplace listings plus the list-initiation flow (List).
type Processor interface {
	GetAll() model.Provider[[]Model]
	GetById(id string) (Model, error)
	Create(m Model) (Model, error)
	Browse(worldId world.Id, state State, f BrowseFilter) ([]Model, error)
	TransitionState(id string, from State, to State) (bool, error)
	UpdateAuction(id string, currentBid uint32, highBidderId uint32, endsAt *time.Time) error
	// Cancel performs the seller's race-safe cancel in ONE local DB transaction:
	// it conditionally transitions the listing active->cancelled and, only when
	// that transition affected exactly one row, creates the seller's holding
	// (origin=cancelled) copying the listing's item snapshot. The conditional
	// transition is the cancel-vs-buy arbiter — a concurrent buy that already
	// moved the row out of active makes this caller the loser (Won=false, no
	// holding created). It is shared by the Kafka cancel handler and the REST
	// DELETE so the tx logic exists once.
	Cancel(id string) (CancelResult, error)
	// Expire runs the same active->holding(seller) transition as Cancel but with
	// the listing moving to expired and the holding recorded with origin=expired.
	// It is the per-listing transition applied by the DB-driven expiration ticker.
	Expire(id string) (CancelResult, error)
	// List validates the request against the tenant config (price floor, active
	// cap, auction duration) and, when valid, pre-allocates a listing id and
	// emits a TransferToMts saga (fee debit + custody transfer). It returns the
	// pre-allocated listing id. It does NOT create the listing row — the row is
	// created only on the custody consumer's AcceptToMtsListing.
	List(req ListRequest) (uuid.UUID, error)
}

type ProcessorImpl struct {
	l       logrus.FieldLogger
	ctx     context.Context
	db      *gorm.DB
	emitter SagaEmitter
}

// Option mutates a ProcessorImpl during construction.
type Option func(*ProcessorImpl)

// WithSagaEmitter overrides the saga emitter (default: the real saga Processor).
// Tests inject a capturing stub so the saga can be asserted without Kafka.
func WithSagaEmitter(e SagaEmitter) Option {
	return func(p *ProcessorImpl) {
		p.emitter = e
	}
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, opts ...Option) Processor {
	p := &ProcessorImpl{l: l, ctx: ctx, db: db}
	p.emitter = saga.NewProcessor(l, ctx)
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *ProcessorImpl) GetAll() model.Provider[[]Model] {
	return model.SliceMap(modelFromEntity)(getAll()(p.db.WithContext(p.ctx)))()
}

func (p *ProcessorImpl) GetById(id string) (Model, error) {
	return GetById(id)(p.db.WithContext(p.ctx))()
}

// Create persists a new listing and returns the stored Model (with its assigned
// surrogate id).
func (p *ProcessorImpl) Create(m Model) (Model, error) {
	return CreateListing(p.db.WithContext(p.ctx), m)
}

// Browse returns the listings for a world filtered by state and the optional
// filter set (category, sub-category, sale type, item id, seller name) with
// pagination. The signature mirrors the getBrowse provider exactly.
func (p *ProcessorImpl) Browse(worldId world.Id, state State, f BrowseFilter) ([]Model, error) {
	return model.SliceMap(modelFromEntity)(getBrowse(worldId, state, f)(p.db.WithContext(p.ctx)))()()
}

// TransitionState performs the race-safe conditional transition, returning true
// iff exactly one row moved from `from` to `to` (the cancel-vs-buy race resolves
// to a single winner; a loser sees zero rows affected and gets false).
func (p *ProcessorImpl) TransitionState(id string, from State, to State) (bool, error) {
	affected, err := UpdateState(p.db.WithContext(p.ctx), id, from, to)
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}

// UpdateAuction updates the live auction fields (current bid, high bidder, and
// the optional end time). Used by the bid path.
func (p *ProcessorImpl) UpdateAuction(id string, currentBid uint32, highBidderId uint32, endsAt *time.Time) error {
	return UpdateAuction(p.db.WithContext(p.ctx), id, currentBid, highBidderId, endsAt)
}

// CancelResult reports the outcome of a Cancel attempt. Won is true iff this
// caller won the cancel-vs-buy race (the conditional transition affected exactly
// one row); on a win the remaining fields describe the created seller holding and
// the listing snapshot so the caller can emit a LISTING_CANCELLED event. On a
// loss (Won=false) no holding was created and the other fields are zero values.
type CancelResult struct {
	Won       bool
	HoldingId uuid.UUID
	SellerId  uint32
	ItemId    uint32
}

// Cancel runs the race-safe active->holding(seller) transition in one transaction.
// See the interface doc for semantics. The conditional UpdateState is the race
// arbiter; composing it with the holding insert in the same ExecuteTransaction
// guarantees the cancel can never half-complete (a cancelled row without its
// seller holding, or vice versa).
func (p *ProcessorImpl) Cancel(id string) (CancelResult, error) {
	return p.transitionToSellerHolding(p.db.WithContext(p.ctx), id, StateCancelled, holding.OriginCancelled)
}

// Expire runs the SAME race-safe active->holding(seller) transition as Cancel but
// records the holding with origin=expired and moves the listing to the expired
// state. It is the local transition the DB-driven expiration ticker applies once
// an auction's ends_at has passed. Sharing transitionToSellerHolding keeps the
// atomic tx logic in one place — Cancel and Expire differ only in the terminal
// listing state and the holding origin.
func (p *ProcessorImpl) Expire(id string) (CancelResult, error) {
	return p.transitionToSellerHolding(p.db.WithContext(p.ctx), id, StateExpired, holding.OriginExpired)
}

// transitionToSellerHolding is the shared, race-safe active->terminal transition
// underpinning both Cancel (terminal=cancelled, origin=cancelled) and Expire
// (terminal=expired, origin=expired). In one ExecuteTransaction it conditionally
// moves the listing out of active and, only when that affected exactly one row,
// creates the seller's holding copying the item snapshot. The conditional
// UpdateState is the race arbiter (a concurrent buy that already moved the row
// makes this caller the loser: Won=false, no holding).
//
// The holding's tenant_id is taken from the listing ROW (lm.TenantId()), not from
// the request context, so the transition is tenant-self-describing: the
// cross-tenant ticker can apply it without reconstructing a tenant model (which
// would require region/version coordinates the listings table does not store).
// The db handed in carries whatever scoping the caller wants — a tenant-scoped
// context for Cancel, a tenant-id-scoped WithoutTenantFilter context for the sweep.
func (p *ProcessorImpl) transitionToSellerHolding(db *gorm.DB, id string, terminal State, origin holding.Origin) (CancelResult, error) {
	var res CancelResult
	terr := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		lm, gerr := GetById(id)(tx)()
		if gerr != nil {
			return gerr
		}

		// Conditional active->terminal transition: the race arbiter. 0 rows means
		// a concurrent buy already won — this caller is the loser.
		affected, uerr := UpdateState(tx, id, StateActive, terminal)
		if uerr != nil {
			return uerr
		}
		if affected != 1 {
			// Race loser: create no holding, leave Won=false.
			return nil
		}

		hm, berr := holding.NewBuilder(lm.TenantId(), lm.WorldId(), lm.SellerId()).
			SetOrigin(origin).
			SetTemplateId(lm.TemplateId()).
			SetQuantity(lm.Quantity()).
			SetStrength(lm.Strength()).
			SetDexterity(lm.Dexterity()).
			SetIntelligence(lm.Intelligence()).
			SetLuck(lm.Luck()).
			SetHP(lm.HP()).
			SetMP(lm.MP()).
			SetWeaponAttack(lm.WeaponAttack()).
			SetMagicAttack(lm.MagicAttack()).
			SetWeaponDefense(lm.WeaponDefense()).
			SetMagicDefense(lm.MagicDefense()).
			SetAccuracy(lm.Accuracy()).
			SetAvoidability(lm.Avoidability()).
			SetHands(lm.Hands()).
			SetSpeed(lm.Speed()).
			SetJump(lm.Jump()).
			SetSlots(lm.Slots()).
			SetLevel(lm.Level()).
			SetItemLevel(lm.ItemLevel()).
			SetItemExp(lm.ItemExp()).
			SetRingId(lm.RingId()).
			SetViciousCount(lm.ViciousCount()).
			SetFlags(lm.Flags()).
			Build()
		if berr != nil {
			return berr
		}
		stored, cerr := holding.CreateHolding(tx, hm)
		if cerr != nil {
			return cerr
		}

		res = CancelResult{
			Won:       true,
			HoldingId: stored.Id(),
			SellerId:  lm.SellerId(),
			ItemId:    lm.TemplateId(),
		}
		return nil
	})
	if terr != nil {
		return CancelResult{}, terr
	}
	return res, nil
}

// List is the server-authoritative list-initiation flow. It validates the
// request against the tenant's MTS configuration and, on success, pre-allocates
// the listing id and emits a TransferToMts saga.
//
// The saga is [AwardMesos(-listingFee), TransferToMts{...}]: the fee debit runs
// first, then the custody transfer (which the orchestrator expands into
// release_from_character + accept_to_mts_listing). The listing row is created in
// `active` only by the custody consumer's AcceptToMtsListing — never here, since
// the item must leave inventory before the listing exists.
func (p *ProcessorImpl) List(req ListRequest) (uuid.UUID, error) {
	t, err := tenant.FromContext(p.ctx)()
	if err != nil {
		return uuid.Nil, err
	}
	cfg := configuration.GetRegistry().GetTenantConfig(p.l, p.ctx, t.Id())

	// Price floor: server-authoritative minimum NX price.
	if req.ListValue < cfg.PriceFloor() {
		return uuid.Nil, fmt.Errorf("list value %d is below the price floor %d", req.ListValue, cfg.PriceFloor())
	}

	// Active-listing cap: a seller may hold at most maxActiveListings active rows.
	count, err := getActiveCountBySeller(req.SellerId)(p.db.WithContext(p.ctx))
	if err != nil {
		return uuid.Nil, err
	}
	if count >= int64(cfg.MaxActiveListings()) {
		return uuid.Nil, fmt.Errorf("seller %d already has %d active listings (cap %d)", req.SellerId, count, cfg.MaxActiveListings())
	}

	// Auction duration: integer hours in [auctionMinHours, auctionMaxHours]. The
	// 1-hour step is implicit — DurationHours is an int, so any whole-hour value
	// in range is accepted and fractional durations are not representable.
	var endsAt *time.Time
	if req.SaleType == SaleTypeAuction {
		if req.DurationHours < cfg.AuctionMinHours() || req.DurationHours > cfg.AuctionMaxHours() {
			return uuid.Nil, fmt.Errorf("auction duration %dh is outside the allowed range [%d, %d]",
				req.DurationHours, cfg.AuctionMinHours(), cfg.AuctionMaxHours())
		}
		end := time.Now().Add(time.Duration(req.DurationHours) * time.Hour)
		endsAt = &end
	}

	// Pre-allocate the listing id so the saga and the eventual AcceptToMtsListing
	// agree on it (deterministic, replay-idempotent creation).
	listingId := uuid.New()
	transactionId := uuid.New()

	builder := saga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(saga.MtsOperation).
		SetInitiatedBy(fmt.Sprintf("character_%d", req.SellerId))

	// Step 1: debit the listing fee (AwardMesos with a negative amount).
	builder.AddStep("award_mesos", saga.Pending, saga.AwardMesos, saga.AwardMesosPayload{
		CharacterId: req.SellerId,
		WorldId:     req.WorldId,
		ActorId:     req.SellerId,
		ActorType:   "SYSTEM",
		Amount:      -int32(cfg.ListingFee()),
		ShowEffect:  false,
	})

	// Step 2: transfer the item into MTS custody. The orchestrator expands this
	// composite into release_from_character + accept_to_mts_listing; the latter
	// creates the listing row in `active`.
	builder.AddStep("transfer_to_mts", saga.Pending, saga.TransferToMts, saga.TransferToMtsPayload{
		TransactionId:       transactionId,
		CharacterId:         req.SellerId,
		WorldId:             req.WorldId,
		SourceInventoryType: req.SourceInventoryType,
		AssetId:             req.AssetId,
		Quantity:            req.Quantity,
		ListingId:           listingId,
		SellerName:          req.SellerName,
		SaleType:            string(req.SaleType),
		ListValue:           req.ListValue,
		BuyNowPrice:         req.BuyNowPrice,
		CommissionRate:      cfg.CommissionRate(),
		Category:            req.Category,
		SubCategory:         req.SubCategory,
		EndsAt:              endsAt,
		MinIncrement:        cfg.MinBidIncrement(),
	})

	// Timeout MUST be set explicitly and scaled for the effective step count.
	// N=2: the fee step counts as 1 and the TransferToMts composite expands to 2
	// (release_from_character + accept_to_mts_listing). A flat timeout rolls back
	// a legitimate multi-step saga under a stressed broker.
	const transferToMtsExpandedSteps = 2 // release_from_character + accept_to_mts_listing
	numSteps := 1 + transferToMtsExpandedSteps
	builder.SetTimeout(listSagaBaseTimeout + time.Duration(numSteps)*listSagaPerStepTimeout)

	if err := p.emitter.Create(builder.Build()); err != nil {
		return uuid.Nil, err
	}
	return listingId, nil
}
