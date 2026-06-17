package listing

import (
	"context"
	"fmt"
	"time"

	"atlas-mts/configuration"
	"atlas-mts/saga"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
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
