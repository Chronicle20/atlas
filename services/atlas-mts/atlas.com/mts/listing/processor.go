package listing

import (
	"context"
	"fmt"
	"math"
	"time"

	"atlas-mts/bid"
	"atlas-mts/configuration"
	"atlas-mts/holding"
	"atlas-mts/saga"
	"atlas-mts/wallet"

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

// BalanceReader abstracts the buyer NX Prepaid balance read so the buy flow can
// be exercised without a live cash-shop wallet. The production implementation is
// the wallet package's Processor (a REST read of atlas-cashshop's wallet); tests
// inject a stub. The read is a best-effort pre-check only — the saga's debit-first
// AwardCurrency step is the authoritative insufficient-funds enforcement.
type BalanceReader interface {
	PrepaidBalance(accountId uint32) (uint32, error)
}

// ListRequest carries the seller-supplied parameters for a TransferToMts list
// initiation. The item reference (assetId + sourceInventoryType) identifies the
// inventory slot to remove; the snapshot itself is looked up by the saga
// expansion, not carried here.
type ListRequest struct {
	WorldId             world.Id
	SellerId            uint32
	SellerAccountId     uint32
	SellerName          string
	SaleType            SaleType
	SourceInventoryType byte
	AssetId             uint32
	Quantity            uint32
	ListValue           uint32
	BuyNowPrice         *uint32
	DurationHours       int    // auction only; hours from now until the auction ends
	MinIncrement        uint32 // auction only; the seller's bid increment (0 => tenant default)
	Category            string
	SubCategory         string
}

// BuyRequest carries the caller-supplied parameters for a buy / buy-now
// settlement. The buyer's id+account come from the channel session; the seller's
// account is caller-supplied too (the channel resolves it at buy time — atlas-mts
// stores only the seller characterId on the listing, and there is no clean
// characterId->accountId resolver outside a session, mirroring how the saga
// AwardCurrencyPayload requires AccountId to be supplied rather than resolved).
// The seller characterId, listValue, and commissionRate are read from the listing
// row, not carried here.
type BuyRequest struct {
	WorldId         world.Id
	ListingId       uuid.UUID
	BuyerId         uint32
	BuyerAccountId  uint32
	SellerAccountId uint32
	// BuyNow distinguishes an immediate-buyout of an auction (BUY_AUCTION_IMM,
	// mode 0x14) from a plain fixed-price buy (BUY, mode 0x10). When true the buy
	// settles at the listing's buyNowPrice (the immediate-buyout price), not its
	// listValue; the listing MUST be an auction carrying a buy-now price. When
	// false the buy settles at listValue (the fixed-price path).
	BuyNow bool
}

// BidRequest carries the caller-supplied parameters for an auction bid. The
// bidder's id+account come from the channel session; the listing's currentBid,
// minIncrement, listValue, and commissionRate are read from the row, never carried
// here. Amount is the raw bid in NX (the escrow holds the MARKED-UP amount).
type BidRequest struct {
	WorldId         world.Id
	ListingId       uuid.UUID
	BidderId        uint32
	BidderAccountId uint32
	Amount          uint32
}

// BidResult reports the outcome of a successful PlaceBid so the caller can emit the
// status events and record history. ItemId/Quantity/SellerId describe the listing;
// HadPrior/PreviousBidderId/PreviousBidAmount describe the displaced high bidder (if
// any) for the OUTBID event and the outbid bidder's bid-lost history row.
type BidResult struct {
	ItemId            uint32
	Quantity          uint32
	SellerId          uint32
	HadPrior          bool
	PreviousBidderId  uint32
	PreviousBidAmount uint32
}

// SettleRequest carries the caller-supplied parameters for the auction
// settle-at-expiry decision. The ticker supplies the winner (the listing's current
// high bidder) plus the account ids needed for the seller credit and the winner's
// holding. When the listing has no high bidder the winner/account fields are zero
// and SettleAuction takes the expire-to-seller path.
type SettleRequest struct {
	ListingId       uuid.UUID
	WorldId         world.Id
	WinnerId        uint32
	WinnerAccountId uint32
	SellerAccountId uint32
}

// SettleResult reports the outcome of a SettleAuction. HadWinner is true iff the
// auction had a held high bid and was settled to the winner (seller credit + custody
// move emitted). Expired is true iff the auction had no bids and was returned to the
// seller's holding via the local Expire transition. Exactly one of the two is true
// on success.
type SettleResult struct {
	HadWinner bool
	Expired   bool
}

// listSagaBaseTimeout and listSagaPerStepTimeout define the step-count-scaled
// timeout for the list saga. The orchestrator processes saga steps serially over
// Kafka, so the timeout budget must grow with the number of effective steps; a
// flat timeout rolls back legitimate multi-step sagas (see the preset-creation
// timeout bug, bug_preset_creation_saga_flat_timeout).
//
// The per-step budget must cover ONE full cross-service Kafka round-trip
// (command -> owning service -> status event -> orchestrator), which under a
// stressed broker is measured in seconds, not ms — an observed MTS buy had a
// single wallet-credit step take ~11s, tripping the old 1s/step budget and
// firing compensation while the step was still in flight (item delivered but
// payment reverted). 15s/step covers the worst case seen with headroom.
const (
	listSagaBaseTimeout    = 10 * time.Second
	listSagaPerStepTimeout = 15 * time.Second
)


// Processor exposes the REST-facing CRUD and state-transition operations over
// marketplace listings plus the list-initiation flow (List).
type Processor interface {
	GetAll() model.Provider[[]Model]
	GetById(id string) (Model, error)
	// GetBySerial resolves a listing by its per-(tenant, world) ITC serial (the
	// client's nITCSN). It is the resolver the channel ITC_OPERATION arms use to
	// translate the wire's uint32 serial into the UUID-keyed listing for the
	// cancel/buy/bid flows.
	GetBySerial(worldId world.Id, sn uint32) (Model, error)
	Create(m Model) (Model, error)
	Browse(worldId world.Id, state State, f BrowseFilter) ([]Model, error)
	CountBrowse(worldId world.Id, state State, f BrowseFilter) (int64, error)
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
	// Buy settles a buy / buy-now against an active listing. It loads the listing
	// (must be active), computes the marked-up price from the listing's captured
	// listValue and commissionRate, pre-checks the buyer's NX Prepaid balance, and
	// — when sufficient — emits a debit-first MtsSettlePurchase saga. It does NOT
	// mutate the listing row or any holding directly: the listing flips to sold and
	// the buyer holding is created only by the orchestrator-driven move step. The
	// commission (markedUp - listValue) is the sink and is never credited.
	Buy(req BuyRequest) error
	// PlaceBid places a bid on an active auction listing. It validates the listing
	// is an active auction and the bid clears the floor (listValue for the first
	// bid, else currentBid + minIncrement), then — under a race-safe compare-and-swap
	// on the listing row — records a held Bid (with a fresh escrow txn id) and
	// advances the listing's currentBid/highBidder. It escrows the MARKED-UP amount
	// (bid * (1 + commissionRate)) by emitting an MtsBidEscrow{-markedUp} saga so the
	// winner's settlement matches buy-now. On an outbid it RELEASES the prior high
	// bidder's escrow (MtsBidEscrow{+markedUpPrior}) and marks their Bid released.
	// It returns a BidResult so the caller can emit BID_PLACED (and, on an outbid,
	// OUTBID) and record the outbid bidder's bid-lost history row.
	PlaceBid(req BidRequest) (BidResult, error)
	// SettleAuction settles an expired auction. With a high bidder it credits the
	// seller's points (+listValue) and moves custody to the winner WITHOUT
	// re-debiting the winner (the winner's prepaid was already escrowed at bid time;
	// re-using MtsSettlePurchase would double-debit), and marks the winning bid won.
	// With NO bids it returns the item to the seller's holding via the local Expire
	// transition (origin=expired). The outcome is reported in SettleResult.
	SettleAuction(req SettleRequest) (SettleResult, error)
}

type ProcessorImpl struct {
	l       logrus.FieldLogger
	ctx     context.Context
	db      *gorm.DB
	emitter SagaEmitter
	balance BalanceReader
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

// WithBalanceReader overrides the buyer-prepaid balance reader (default: the real
// wallet Processor, a REST read of atlas-cashshop's wallet). Tests inject a stub
// so the insufficient/sufficient-funds pre-check can be asserted without Kafka or
// a live cash-shop.
func WithBalanceReader(b BalanceReader) Option {
	return func(p *ProcessorImpl) {
		p.balance = b
	}
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, opts ...Option) Processor {
	p := &ProcessorImpl{l: l, ctx: ctx, db: db}
	p.emitter = saga.NewProcessor(l, ctx)
	p.balance = wallet.NewProcessor(l, ctx)
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

// GetBySerial resolves a listing by its per-(tenant, world) ITC serial.
func (p *ProcessorImpl) GetBySerial(worldId world.Id, sn uint32) (Model, error) {
	return GetBySerial(worldId, sn)(p.db.WithContext(p.ctx))()
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

// CountBrowse returns the TOTAL number of listings matching the same filters
// Browse applies, ignoring paging — so a caller can report a real total and
// last page rather than inferring them from one page's length.
func (p *ProcessorImpl) CountBrowse(worldId world.Id, state State, f BrowseFilter) (int64, error) {
	return countBrowse(worldId, state, f)(p.db.WithContext(p.ctx))
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
	// Held bid escrow to release: set when a cancelled auction still had an open
	// high bid at cancel time (only the current high bidder still holds escrow —
	// outbid bidders were released as they were outbid). A zero HeldBidderId means
	// there was nothing to release. The caller (Cancel) reverses the hold.
	HeldBidderId        uint32
	HeldBidderAccountId uint32
	HeldBidAmount       uint32
}

// Cancel runs the race-safe active->holding(seller) transition in one transaction.
// See the interface doc for semantics. The conditional UpdateState is the race
// arbiter; composing it with the holding insert in the same ExecuteTransaction
// guarantees the cancel can never half-complete (a cancelled row without its
// seller holding, or vice versa). If the cancelled auction still had an open high
// bid, its escrow is released (reversing the hold) via a single-step saga.
func (p *ProcessorImpl) Cancel(id string) (CancelResult, error) {
	res, err := p.transitionToSellerHolding(p.db.WithContext(p.ctx), id, StateCancelled, holding.OriginCancelled)
	if err != nil {
		return res, err
	}
	if res.HeldBidderId != 0 {
		releaseTxnId := uuid.New()
		rb := saga.NewBuilder().
			SetTransactionId(releaseTxnId).
			SetSagaType(saga.MtsOperation).
			SetInitiatedBy(fmt.Sprintf("character_%d", res.HeldBidderId))
		rb.AddStep("mts_bid_escrow_release", saga.Pending, saga.MtsBidEscrow, saga.MtsBidEscrowPayload{
			TransactionId:   releaseTxnId,
			ListingId:       uuid.MustParse(id),
			BidderId:        res.HeldBidderId,
			BidderAccountId: res.HeldBidderAccountId,
			Amount:          int32(res.HeldBidAmount),
		})
		rb.SetTimeout(bidEscrowTimeout())
		if eerr := p.emitter.Create(rb.Build()); eerr != nil {
			return res, eerr
		}
	}
	return res, nil
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

		// If the auction still had an open high bid, mark that held bid released in
		// THIS tx and surface it so the caller reverses the escrow hold. Only the
		// current high bidder still holds escrow (outbid bidders were released as
		// they were outbid). Expire only runs on no-bid auctions, so this is a no-op
		// there; it matters for Cancel of an auction with a live bid.
		if lm.SaleType() == SaleTypeAuction && lm.HighBidderId() != 0 {
			heldId, heldAccount, gerr := heldBidFor(tx, lm.Id(), lm.HighBidderId())
			if gerr != nil {
				return gerr
			}
			if heldId != uuid.Nil {
				if _, uerr := bid.UpdateState(tx, heldId.String(), bid.StateHeld, bid.StateReleased); uerr != nil {
					return uerr
				}
				res.HeldBidderId = lm.HighBidderId()
				res.HeldBidderAccountId = heldAccount
				res.HeldBidAmount = lm.CurrentBid()
			}
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
// minIncrementOrDefault returns the seller-supplied bid increment when non-zero,
// else the tenant default. The register-auction packet carries the seller's chosen
// increment; a 0 (older channels / non-auction paths) falls back to config.
func minIncrementOrDefault(supplied uint32, def uint32) uint32 {
	if supplied != 0 {
		return supplied
	}
	return def
}

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
	//
	// Fixed sales carry a sale term too (era-faithful: the original MTS
	// expired fixed listings back to the seller after their term). The client
	// sends no duration for REGISTER_SALE, so the term is the tenant's
	// fixedSaleHours knob (default 168h). The sweep's no-bids arm returns the
	// expired listing to the seller holding (origin=expired), same as an
	// unsold auction.
	var endsAt *time.Time
	if req.SaleType == SaleTypeAuction {
		if req.DurationHours < cfg.AuctionMinHours() || req.DurationHours > cfg.AuctionMaxHours() {
			return uuid.Nil, fmt.Errorf("auction duration %dh is outside the allowed range [%d, %d]",
				req.DurationHours, cfg.AuctionMinHours(), cfg.AuctionMaxHours())
		}
		end := time.Now().Add(time.Duration(req.DurationHours) * time.Hour)
		endsAt = &end
	} else {
		end := time.Now().Add(time.Duration(cfg.FixedSaleDurationHours()) * time.Hour)
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

	// Step 1: debit the registration fee in NX. The client previews (and the
	// flat meso fee charged to the seller to create a listing. (The 500 NX + 7%
	// the in-game guide describes is the BUYER's commission at purchase, not this
	// seller listing fee — see Buy's markedUp.)
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
		SellerAccountId:     req.SellerAccountId,
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
		MinIncrement:        minIncrementOrDefault(req.MinIncrement, cfg.MinBidIncrement()),
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

// buySagaBaseTimeout and buySagaPerStepTimeout mirror the list flow's
// step-count-scaled timeout. The MtsSettlePurchase composite expands to N=3
// effective steps in the orchestrator (award_currency buyer, award_currency
// seller, mts_move_listing_to_holding), so the budget must grow with that count;
// a flat (or too-tight) timeout rolls back a legitimate multi-step saga under a
// stressed broker (see bug_preset_creation_saga_flat_timeout). The per-step
// budget covers one full cross-service Kafka round-trip — seconds under load,
// not ms; see the list-flow constants for the incident that set 15s.
const (
	buySagaBaseTimeout    = 10 * time.Second
	buySagaPerStepTimeout = 15 * time.Second
)

// Buy settles a buy / buy-now against an active listing. The flow is:
//
//  1. Load the listing; it MUST be active (a sold/cancelled/expired listing is
//     rejected — the buy cannot proceed). The seller characterId, listValue, and
//     commissionRate are taken from the row (server-authoritative, captured at
//     list time), never from the caller.
//  2. The listing's listValue/buyNowPrice are ALREADY market (commission-
//     inclusive) prices — the commission was baked in once at list time (see the
//     custody consumer's AcceptToMtsListing). So priceBasis IS the price the buyer
//     pays; there is no second markup here.
//  3. Pre-check the buyer's NX Prepaid balance >= priceBasis. This is a
//     best-effort gate that fails fast on an obviously under-funded buy; the
//     saga's debit-first AwardCurrency step is the AUTHORITATIVE enforcement (if
//     the balance changed between the read and the debit, the first saga step
//     fails, the saga aborts, and nothing is granted or moved).
//  4. Emit a single MtsSettlePurchase composite. The orchestrator expands it
//     debit-first: award_currency(buyer prepaid, -priceBasis) FIRST so a mid-saga
//     failure grants nothing, then award_currency(seller points,
//     +UnMarkUp(priceBasis)), then mts_move_listing_to_holding(buyer). The item
//     lands in the buyer's mts holding (origin=purchased), NEVER inventory.
//     Commission = priceBasis - UnMarkUp(priceBasis) is never credited to anyone
//     — it is the sink.
//
// Buy does NOT mutate the listing row or create any holding directly; those
// effects happen only via the orchestrator-driven move step (which marks the
// listing sold and creates the buyer holding in one atlas-mts local tx).
func (p *ProcessorImpl) Buy(req BuyRequest) error {
	lm, err := GetById(req.ListingId.String())(p.db.WithContext(p.ctx))()
	if err != nil {
		return fmt.Errorf("load listing %s: %w", req.ListingId, err)
	}
	if lm.State() != StateActive {
		return fmt.Errorf("listing %s is not active (state=%s); cannot buy: %w", req.ListingId, lm.State(), ErrListingUnavailable)
	}

	// Price basis: a plain buy settles at the listing's listValue (the fixed-price
	// value). A buy-now (BUY_AUCTION_IMM) settles at the listing's buyNowPrice — the
	// immediate-buyout price — which is only meaningful on an auction that carries
	// one. Both are MARKET (commission-inclusive) prices, so priceBasis is exactly
	// what the buyer pays; the seller credit is derived via UnMarkUp below.
	priceBasis := lm.ListValue()
	if req.BuyNow {
		if lm.SaleType() != SaleTypeAuction || lm.BuyNowPrice() == nil {
			return fmt.Errorf("listing %s is not a buy-now auction (saleType=%s, buyNow=%v); cannot buy-now", req.ListingId, lm.SaleType(), lm.BuyNowPrice())
		}
		priceBasis = *lm.BuyNowPrice()
	}
	t, terr := tenant.FromContext(p.ctx)()
	if terr != nil {
		return terr
	}
	cfg := configuration.GetRegistry().GetTenantConfig(p.l, p.ctx, t.Id())

	// Best-effort pre-check; the saga's first (debit) step is the authoritative gate.
	prepaid, err := p.balance.PrepaidBalance(req.BuyerAccountId)
	if err != nil {
		return fmt.Errorf("read buyer %d prepaid balance: %w", req.BuyerAccountId, err)
	}
	if prepaid < priceBasis {
		return fmt.Errorf("buyer %d prepaid %d is below the market price %d: %w", req.BuyerAccountId, prepaid, priceBasis, ErrInsufficientPrepaid)
	}

	transactionId := uuid.New()

	builder := saga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(saga.MtsOperation).
		SetInitiatedBy(fmt.Sprintf("character_%d", req.BuyerId))

	// One composite step: MtsSettlePurchase. The orchestrator owns the debit-first
	// ordering of the expansion (buyer prepaid -priceBasis, seller points
	// +UnMarkUp(priceBasis), move-to-holding). Commission is never credited: the
	// payload only ever carries the un-marked-up base as the seller credit, so the
	// difference stays the sink.
	builder.AddStep("mts_settle_purchase", saga.Pending, saga.MtsSettlePurchase, saga.MtsSettlePurchasePayload{
		TransactionId:   transactionId,
		ListingId:       req.ListingId,
		WorldId:         req.WorldId,
		BuyerId:         req.BuyerId,
		BuyerAccountId:  req.BuyerAccountId,
		SellerId:        lm.SellerId(),
		SellerAccountId: req.SellerAccountId,
		MarkedUpPrice:   int32(priceBasis),
		ListValue:       int32(UnMarkUp(priceBasis, lm.CommissionRate(), cfg.CommissionBase())),
	})

	// Timeout MUST be set explicitly and scaled for N=3: the MtsSettlePurchase
	// composite expands to award_currency x2 + mts_move_listing_to_holding.
	const settleExpandedSteps = 3 // award_currency(buyer) + award_currency(seller) + mts_move_listing_to_holding
	builder.SetTimeout(buySagaBaseTimeout + time.Duration(settleExpandedSteps)*buySagaPerStepTimeout)

	return p.emitter.Create(builder.Build())
}

// MarkedUp returns ceil(amount * (1 + commissionRate)) + commissionBase — the
// market (commission-inclusive) price derived from a base amount. It mirrors the
// list-time markup rule: round UP so the fractional NX falls toward the sink.
// Exported so the custody consumer can mark up the seller-supplied base prices
// ONCE at listing-creation time (the new commission-inclusive pricing model
// stores/transacts everything in market units thereafter).
func MarkedUp(amount uint32, commissionRate float64, commissionBase uint32) uint32 {
	return uint32(math.Ceil(float64(amount)*(1.0+commissionRate))) + commissionBase
}

// UnMarkUp inverts MarkedUp: given a market (commission-inclusive) price, it
// returns the base amount the seller nets (the platform keeps the difference as
// commission). If market does not exceed the flat commissionBase there is no
// base left for the seller (0). This is an approximate inverse of the ceil-based
// MarkedUp (float division, truncated to uint32), used at settlement time to
// compute the seller's credit from a stored market price.
func UnMarkUp(market uint32, commissionRate float64, commissionBase uint32) uint32 {
	if market <= commissionBase {
		return 0
	}
	return uint32(float64(market-commissionBase) / (1.0 + commissionRate))
}

// bidEscrowTimeout scales the single-step escrow saga's timeout. MtsBidEscrow is a
// single-step wallet adjust (N=1): the orchestrator routes it straight to the
// cash-shop wallet without expansion. A flat timeout is still wrong under a stressed
// broker, so it is scaled for N=1 (base + 1*perStep), mirroring the list/buy flows.
func bidEscrowTimeout() time.Duration {
	const escrowSteps = 1
	return buySagaBaseTimeout + time.Duration(escrowSteps)*buySagaPerStepTimeout
}

// PlaceBid is the server-authoritative auction-bid flow. See the interface doc for
// semantics. The flow is:
//
//  1. Load the listing; it MUST be an active auction (a fixed-price, sold,
//     cancelled, or expired listing is rejected).
//  2. Validate the floor: the FIRST bid (no high bidder) must clear listValue; a
//     subsequent bid must clear currentBid + minIncrement. The thresholds are read
//     from the row, never from the caller.
//  3. In ONE local DB transaction: a race-safe compare-and-swap (AdvanceAuctionBid)
//     advances the listing's currentBid/highBidder only if the row is still active
//     with the prior bid the caller read — the optimistic-concurrency arbiter. On a
//     lost race (0 rows) nothing is recorded and the caller is rejected (the
//     concurrent bid won). On a win, a held Bid is recorded with a fresh escrow txn
//     id, and — if there was a prior high bidder — that bidder's held Bid is marked
//     released in the same tx.
//  4. Emit MtsBidEscrow{-req.Amount} to HOLD the new bidder's prepaid. req.Amount
//     is already a MARKET (commission-inclusive) bid — the listing's listValue/
//     currentBid are market prices too, so the hold is the bid AS-IS, no second
//     markup. On an outbid, emit a second MtsBidEscrow{+priorBid} to RELEASE the
//     prior bidder's escrow — the exact raw amount that was held for their bid.
//
// PlaceBid escrows the bid amount as-is and records it on the Bid row too (both
// are the same market figure now). The escrow keys off the Bid's escrowTxnId so a
// release reverses the exact hold.
func (p *ProcessorImpl) PlaceBid(req BidRequest) (BidResult, error) {
	lm, err := GetById(req.ListingId.String())(p.db.WithContext(p.ctx))()
	if err != nil {
		return BidResult{}, fmt.Errorf("load listing %s: %w", req.ListingId, err)
	}
	if lm.State() != StateActive {
		return BidResult{}, fmt.Errorf("listing %s is not active (state=%s); cannot bid: %w", req.ListingId, lm.State(), ErrListingUnavailable)
	}
	if lm.SaleType() != SaleTypeAuction {
		return BidResult{}, fmt.Errorf("listing %s is not an auction (saleType=%s); cannot bid", req.ListingId, lm.SaleType())
	}

	priorBid := lm.CurrentBid()
	priorBidder := lm.HighBidderId()
	hasPrior := priorBidder != 0

	// Floor: first bid clears listValue; subsequent clears currentBid + minIncrement.
	var floor uint32
	if hasPrior {
		floor = priorBid + lm.MinIncrement()
	} else {
		floor = lm.ListValue()
	}
	if req.Amount < floor {
		return BidResult{}, fmt.Errorf("bid %d is below the floor %d for listing %s", req.Amount, floor, req.ListingId)
	}

	escrowTxnId := uuid.New()
	t := tenant.MustFromContext(p.ctx)

	// One local tx: the CAS advance (race arbiter) + record the held bid + mark the
	// prior bid released. Composing them guarantees the bid can never half-commit
	// (an advanced listing without its held bid, or an outbid without the prior
	// release mark). priorAccount is the prior high bidder's account id, read from
	// THEIR held Bid row so the outbid release credits the correct wallet (the
	// channel does not carry the prior bidder's account on this request).
	var won bool
	var priorAccount uint32
	terr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		affected, aerr := AdvanceAuctionBid(tx, req.ListingId.String(), priorBid, priorBidder, req.Amount, req.BidderId)
		if aerr != nil {
			return aerr
		}
		if affected != 1 {
			// Lost the compare-and-swap: a concurrent bid already advanced the row.
			return nil
		}
		won = true

		bm, berr := bid.NewBuilder(t.Id(), req.ListingId, req.BidderId).
			SetId(uuid.New()).
			SetBidderAccountId(req.BidderAccountId).
			SetAmount(req.Amount).
			SetEscrowTxnId(escrowTxnId).
			SetState(bid.StateHeld).
			Build()
		if berr != nil {
			return berr
		}
		if _, cerr := bid.CreateBid(tx, bm); cerr != nil {
			return cerr
		}

		// Outbid: mark the prior high bidder's held bid released (capturing their
		// account id for the release credit) so the released-state escrow set stays
		// consistent with the +amount release emitted below.
		if hasPrior {
			prevHeld, paccount, gerr := heldBidFor(tx, req.ListingId, priorBidder)
			if gerr != nil {
				return gerr
			}
			if prevHeld != uuid.Nil {
				priorAccount = paccount
				if _, uerr := bid.UpdateState(tx, prevHeld.String(), bid.StateHeld, bid.StateReleased); uerr != nil {
					return uerr
				}
			}
		}
		return nil
	})
	if terr != nil {
		return BidResult{}, terr
	}
	if !won {
		return BidResult{}, fmt.Errorf("bid for listing %s lost the high-bid race (current bid advanced); rejected", req.ListingId)
	}

	// HOLD the new bidder's prepaid by the bid amount AS-IS (single-step saga, N=1).
	// req.Amount is already a market (commission-inclusive) price — no second markup.
	holdTxnId := escrowTxnId
	holdBuilder := saga.NewBuilder().
		SetTransactionId(holdTxnId).
		SetSagaType(saga.MtsOperation).
		SetInitiatedBy(fmt.Sprintf("character_%d", req.BidderId))
	holdBuilder.AddStep("mts_bid_escrow_hold", saga.Pending, saga.MtsBidEscrow, saga.MtsBidEscrowPayload{
		TransactionId:   holdTxnId,
		ListingId:       req.ListingId,
		BidderId:        req.BidderId,
		BidderAccountId: req.BidderAccountId,
		Amount:          -int32(req.Amount),
	})
	holdBuilder.SetTimeout(bidEscrowTimeout())
	if err := p.emitter.Create(holdBuilder.Build()); err != nil {
		return BidResult{}, err
	}

	// On an outbid, RELEASE the prior bidder's escrow by the SAME raw amount that
	// was held for THEIR bid — a separate single-step saga (N=1). The release
	// exactly reverses the prior hold (no markup on either side now).
	if hasPrior {
		releaseTxnId := uuid.New()
		releaseBuilder := saga.NewBuilder().
			SetTransactionId(releaseTxnId).
			SetSagaType(saga.MtsOperation).
			SetInitiatedBy(fmt.Sprintf("character_%d", priorBidder))
		releaseBuilder.AddStep("mts_bid_escrow_release", saga.Pending, saga.MtsBidEscrow, saga.MtsBidEscrowPayload{
			TransactionId:   releaseTxnId,
			ListingId:       req.ListingId,
			BidderId:        priorBidder,
			BidderAccountId: priorAccount,
			Amount:          int32(priorBid),
		})
		releaseBuilder.SetTimeout(bidEscrowTimeout())
		if err := p.emitter.Create(releaseBuilder.Build()); err != nil {
			return BidResult{}, err
		}
	}

	return BidResult{
		ItemId:            lm.TemplateId(),
		Quantity:          lm.Quantity(),
		SellerId:          lm.SellerId(),
		HadPrior:          hasPrior,
		PreviousBidderId:  priorBidder,
		PreviousBidAmount: priorBid,
	}, nil
}

// auctionSettleTimeout scales the auction-settle saga's timeout. The settle is a
// TWO-step saga (N=2): award_currency(seller points +listValue) +
// mts_move_listing_to_holding(winner). It is NOT MtsSettlePurchase — that debits
// the buyer FIRST, which would DOUBLE-DEBIT the winner whose prepaid was already
// escrowed at bid time. So the buyer-debit step is deliberately omitted here.
func auctionSettleTimeout() time.Duration {
	const settleSteps = 2 // award_currency(seller) + mts_move_listing_to_holding
	return buySagaBaseTimeout + time.Duration(settleSteps)*buySagaPerStepTimeout
}

// SettleAuction settles an expired auction. See the interface doc for semantics.
//
// Currency correctness (the crux): the winner's prepaid was ALREADY debited at bid
// time (the MtsBidEscrow{-currentBid} hold, currentBid being a market price — no
// markup). At settle the seller must be credited the WINNING bid's base
// (UnMarkUp(currentBid)) and the item moved to the winner — the winner is NOT
// re-debited. So the settle saga is exactly:
//
//  1. award_currency(seller account, currencyType=2 points, +UnMarkUp(currentBid))
//  2. mts_move_listing_to_holding(winner)  // marks listing sold + winner holding
//
// The commission (currentBid - UnMarkUp(currentBid)) stays as the sink: the
// winner's hold was the raw winning bid, only the un-marked-up base flows to the
// seller, and the difference is never credited to anyone. Reusing
// MtsSettlePurchase would inject a buyer-debit step (prepaid -currentBid) and
// double-debit the winner — so it is deliberately NOT used.
//
// The seller credit is derived from the FINAL winning bid (lm.CurrentBid()), not
// the starting listValue — the auction may have sold for more than its opening
// price and the seller must be paid off the actual sale price.
//
// With NO high bidder the auction returns to the seller's holding via the local
// Expire transition (origin=expired) and no money-mover is emitted.
func (p *ProcessorImpl) SettleAuction(req SettleRequest) (SettleResult, error) {
	lm, err := GetById(req.ListingId.String())(p.db.WithContext(p.ctx))()
	if err != nil {
		return SettleResult{}, fmt.Errorf("load listing %s: %w", req.ListingId, err)
	}

	// No bids: return the item to the seller via the existing expire transition.
	if lm.HighBidderId() == 0 {
		res, eerr := p.Expire(req.ListingId.String())
		if eerr != nil {
			return SettleResult{}, eerr
		}
		return SettleResult{Expired: res.Won}, nil
	}

	winner := lm.HighBidderId()
	winningBid := lm.CurrentBid()

	// The tenant id is taken from the listing ROW (lm.TenantId()), not from
	// tenant.FromContext(p.ctx): SettleAuction is invoked by the cross-tenant
	// expiration sweep under database.WithoutTenantFilter, whose context carries
	// no reconstructable tenant.Model (see the Sweep doc comment). Reading the
	// commissionBase off the row's own tenant id keeps this call
	// tenant-self-describing, mirroring transitionToSellerHolding.
	cfg := configuration.GetRegistry().GetTenantConfig(p.l, p.ctx, lm.TenantId())
	sellerCredit := UnMarkUp(winningBid, lm.CommissionRate(), cfg.CommissionBase())

	// The seller-points credit is keyed by the seller's cash-shop account id. A
	// zero account is the "not resolved" sentinel — crediting account 0 would be a
	// silent wrong-wallet bug, so reject rather than emit a settle that mis-credits.
	// The seller account is captured onto the listing at list time (SellerAccountId);
	// the ticker reads it from the row.
	if req.SellerAccountId == 0 {
		return SettleResult{}, fmt.Errorf("auction %s has a winner but the seller account is unresolved; cannot credit", req.ListingId)
	}

	// Re-discovery guard (the double-credit money bug): the DB-driven sweep
	// discovers expired auctions by (state='active' AND ends_at<now). The listing
	// only flips out of `active` LATER, in the async MtsMoveListingToHolding step,
	// so a second sweep tick firing before that completes would re-discover this row
	// and emit a SECOND seller credit. To make the row non-discoverable the instant
	// the settle is decided, transition it active->settling under a race-safe CAS in
	// the SAME tx that marks the winning bid won. If the CAS affects 0 rows, another
	// settle (a prior tick or a concurrent one) already claimed this auction — this
	// caller emits nothing (HadWinner=false). settling is excluded from the sweep's
	// discovery set, and MtsMoveListingToHolding transitions settling->sold.
	var claimed bool
	terr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		affected, uerr := UpdateState(tx, req.ListingId.String(), StateActive, StateSettling)
		if uerr != nil {
			return uerr
		}
		if affected != 1 {
			// Lost the settle CAS: a prior/concurrent settle already claimed this
			// auction. Do not mark the bid won or emit — that settle owns the credit.
			return nil
		}
		claimed = true

		// Mark the winning held bid won (race-safe conditional). It is harmless if a
		// concurrent path already moved it; the credit/move below are saga-driven and
		// idempotent on the listing row's settling->sold transition.
		heldId, _, gerr := heldBidFor(tx, req.ListingId, winner)
		if gerr != nil {
			return gerr
		}
		if heldId != uuid.Nil {
			if _, berr := bid.UpdateState(tx, heldId.String(), bid.StateHeld, bid.StateWon); berr != nil {
				return berr
			}
		}
		return nil
	})
	if terr != nil {
		return SettleResult{}, terr
	}
	if !claimed {
		// Another settle won the CAS; this caller settles nothing and re-credits no
		// one. The sweep treats this as a no-op (HadWinner=false, Expired=false).
		return SettleResult{}, nil
	}

	transactionId := uuid.New()
	builder := saga.NewBuilder().
		SetTransactionId(transactionId).
		SetSagaType(saga.MtsOperation).
		SetInitiatedBy(fmt.Sprintf("character_%d", winner))

	// Step 1: credit the seller's points by UnMarkUp(winningBid) — the seller's base
	// off the FINAL winning bid (NO buyer debit — the winner was already debited at
	// bid time).
	builder.AddStep("award_currency_seller", saga.Pending, saga.AwardCurrency, saga.AwardCurrencyPayload{
		CharacterId:  lm.SellerId(),
		AccountId:    req.SellerAccountId,
		CurrencyType: saga.CurrencyTypePoints,
		Amount:       int32(sellerCredit),
	})
	// Step 2: move custody to the winner's holding (marks listing active->sold).
	builder.AddStep("mts_move_listing_to_holding", saga.Pending, saga.MtsMoveListingToHolding, saga.MtsMoveListingToHoldingPayload{
		TransactionId: transactionId,
		ListingId:     req.ListingId,
		BuyerId:       winner,
		WorldId:       req.WorldId,
	})
	builder.SetTimeout(auctionSettleTimeout())

	if err := p.emitter.Create(builder.Build()); err != nil {
		// The settle saga never emitted, so the row would be stranded in `settling`
		// (out of the sweep's discovery set) and never retried. Revert it
		// settling->active under a CAS so the NEXT sweep tick re-discovers and
		// re-settles it. The winning bid's won mark is left as-is (the re-settle's
		// idempotent heldBidFor/UpdateState handles an already-won bid harmlessly).
		if _, rerr := UpdateState(p.db.WithContext(p.ctx), req.ListingId.String(), StateSettling, StateActive); rerr != nil {
			p.l.WithError(rerr).Errorf("failed to revert listing %s settling->active after a settle-emit failure; it may be stranded out of the sweep", req.ListingId)
		}
		return SettleResult{}, err
	}
	return SettleResult{HadWinner: true}, nil
}

// heldBidFor returns the surrogate id AND the stored account id of the single held
// bid placed by bidderId on the listing, or (uuid.Nil, 0) if none. It scans the
// listing's bids (the (tenant_id, listing_id, state) index backs this) and matches
// the held bid for the bidder. The account id is needed so an outbid release credits
// the prior bidder's correct wallet.
func heldBidFor(db *gorm.DB, listingId uuid.UUID, bidderId uint32) (uuid.UUID, uint32, error) {
	bids, err := bid.GetByListingId(listingId)(db)()
	if err != nil {
		return uuid.Nil, 0, err
	}
	for _, b := range bids {
		if b.BidderId() == bidderId && b.State() == bid.StateHeld {
			return b.Id(), b.BidderAccountId(), nil
		}
	}
	return uuid.Nil, 0, nil
}
