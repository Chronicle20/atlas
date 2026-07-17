package listing

import (
	"atlas-mts/configuration"
	"atlas-mts/holding"
	"atlas-mts/transaction"
	"encoding/binary"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// AcceptRequest carries the listing snapshot the custody AcceptToMtsListing
// command applies. It mirrors the command body's fields so the server-authoritative
// row-create business logic (idempotency, category/subCategory derivation, auction
// currentBid seeding, and the full builder assembly) lives in the processor rather
// than the consumer.
type AcceptRequest struct {
	ListingId       uuid.UUID
	WorldId         byte
	SellerId        uint32
	SellerAccountId uint32
	SellerName      string
	SaleType        string

	// item snapshot
	TemplateId uint32
	Quantity   uint32

	// equip stat block
	Strength      uint16
	Dexterity     uint16
	Intelligence  uint16
	Luck          uint16
	HP            uint16
	MP            uint16
	WeaponAttack  uint16
	MagicAttack   uint16
	WeaponDefense uint16
	MagicDefense  uint16
	Accuracy      uint16
	Avoidability  uint16
	Hands         uint16
	Speed         uint16
	Jump          uint16
	Slots         uint16
	Level         byte
	ItemLevel     byte
	ItemExp       uint32
	RingId        uint32
	ViciousCount  uint32
	Flags         uint16

	// sale params
	ListValue      uint32
	BuyNowPrice    *uint32
	CommissionRate float64
	Category       string
	SubCategory    string
	EndsAt         *time.Time
	MinIncrement   uint32

	// offer link
	OfferWishSerial  uint32
	OfferWishOwnerId uint32
}

// Accept CREATES the listing row in active state from the carried snapshot, using
// the caller-supplied ListingId so the create is deterministic and idempotent. A
// replayed delivery (same ListingId) finds the row already present and is a no-op.
// The whole row-create runs in one local DB transaction. The Kafka acks
// (ACCEPTED + LISTING_CREATED) stay in the consumer.
func (p *ProcessorImpl) Accept(req AcceptRequest) error {
	ctx := p.ctx
	b := req
	tdb := p.db.WithContext(ctx)

	return database.ExecuteTransaction(tdb, func(tx *gorm.DB) error {
		// Idempotency: if a row already exists for this listing id, the
		// command has already been applied — no-op, do not create a
		// duplicate.
		if existing, gerr := GetById(b.ListingId.String())(tx)(); gerr == nil && existing.Id() == b.ListingId {
			return nil
		}

		t := tenant.MustFromContext(ctx)
		tid := t.Id()

		// The GET_ITC_LIST browse filters listings by (category, subCategory),
		// which mirror the client's browse "tab" and "type":
		//   category    = the marketplace SECTION / top tab: "1" For Sale
		//                 (fixed-price), "3" Auction. (Sections 2/4/5 — wanted,
		//                 my-page/cart — hold no sale listings.)
		//   subCategory = the item's inventory category (1=equip 2=use 3=setup
		//                 4=etc 5=cash), derived from the templateId.
		// So a fixed USE listing surfaces only under For Sale -> Use.
		category := "1"
		if b.SaleType == string(SaleTypeAuction) {
			category = "3"
		}
		subCategory := b.SubCategory
		if it, ok := inventory.TypeFromItemId(item.Id(b.TemplateId)); ok {
			subCategory = strconv.Itoa(int(it))
		}

		// Auctions seed currentBid to (listValue - increment) so the client's
		// first bid — always current_bid + increment — lands on the seller's
		// starting price (listValue). Without this the first valid bid would be
		// listValue+increment, one increment above the advertised opening price.
		// A listValue not exceeding the increment seeds 0 (no headroom to
		// subtract). Fixed sales have no bid, so it stays 0.
		inc := b.MinIncrement
		if inc == 0 {
			inc = 1
		}
		var currentBid uint32
		if b.SaleType == string(SaleTypeAuction) {
			if b.ListValue > inc {
				currentBid = b.ListValue - inc
			} else {
				currentBid = 0
			}
		}

		m, berr := NewBuilder(tid, world.Id(b.WorldId), b.SellerId).
			SetId(b.ListingId).
			SetSellerAccountId(b.SellerAccountId).
			SetSellerName(b.SellerName).
			SetSaleType(SaleType(b.SaleType)).
			SetState(StateActive).
			SetTemplateId(b.TemplateId).
			SetQuantity(b.Quantity).
			SetStrength(b.Strength).
			SetDexterity(b.Dexterity).
			SetIntelligence(b.Intelligence).
			SetLuck(b.Luck).
			SetHP(b.HP).
			SetMP(b.MP).
			SetWeaponAttack(b.WeaponAttack).
			SetMagicAttack(b.MagicAttack).
			SetWeaponDefense(b.WeaponDefense).
			SetMagicDefense(b.MagicDefense).
			SetAccuracy(b.Accuracy).
			SetAvoidability(b.Avoidability).
			SetHands(b.Hands).
			SetSpeed(b.Speed).
			SetJump(b.Jump).
			SetSlots(b.Slots).
			SetLevel(b.Level).
			SetItemLevel(b.ItemLevel).
			SetItemExp(b.ItemExp).
			SetRingId(b.RingId).
			SetViciousCount(b.ViciousCount).
			SetFlags(b.Flags).
			SetListValue(b.ListValue).
			SetBuyNowPrice(b.BuyNowPrice).
			SetCommissionRate(b.CommissionRate).
			SetCategory(category).
			SetSubCategory(subCategory).
			SetEndsAt(b.EndsAt).
			SetMinIncrement(b.MinIncrement).
			SetCurrentBid(currentBid).
			SetOfferWishSerial(b.OfferWishSerial).
			SetOfferWishOwnerId(b.OfferWishOwnerId).
			Build()
		if berr != nil {
			return berr
		}
		_, cerr := CreateListing(tx, m)
		return cerr
	})
}

// MoveHoldingId derives a deterministic surrogate id for the buyer's holding from
// the (listingId, buyerId) pair. A replayed settlement-move therefore targets the
// same holding id, so the existence-check below short-circuits and no second
// holding is created (mirrors the AcceptToMtsListing id-existence idempotency).
func MoveHoldingId(listingId uuid.UUID, buyerId uint32) uuid.UUID {
	var buf [20]byte
	copy(buf[:16], listingId[:])
	binary.BigEndian.PutUint32(buf[16:], buyerId)
	return uuid.NewSHA1(uuid.Nil, buf[:])
}

// SettleMoveRequest carries the settle-move parameters the custody
// MtsMoveListingToHolding command applies.
type SettleMoveRequest struct {
	ListingId uuid.UUID
	BuyerId   uint32
	WorldId   byte
	// Price is the seller's BASE sale price for THIS settlement, taken from the settle
	// saga payload (the buy flow's price basis): the buy-now price for a buy-now, the
	// list value for a fixed sale, the winning bid for an auction settle. It drives
	// both parties' history rows — re-deriving it from the listing row recorded the
	// last BID for a buy-now auction instead of the buy-now price (task-102 live find).
	Price uint32
}

// SettleMoveResult reports what the consumer needs after the settle-move tx
// commits: the sold item id + seller for the LISTING_SOLD notice, the sale type +
// fulfilled want-ad serial for the post-commit offer side-effects, and the buyer
// holding id for the MOVED ack.
type SettleMoveResult struct {
	HoldingId           uuid.UUID
	ItemId              uint32
	SellerId            uint32
	SoldSaleType        string
	SoldOfferWishSerial uint32
}

// SettleMove settles a purchase: in ONE local DB transaction it (a) loads the
// listing, (b) conditionally marks it sold via UpdateState(active→sold, else
// settling→sold), (c) enforces the single-custody race guard, and (d) creates the
// buyer's holding row (origin=purchased) plus both parties' history rows. Idempotency:
// the buyer holding id is derived deterministically from (listingId, buyerId); a
// replayed delivery finds that holding already present and is a no-op. The
// conditional UpdateState affecting 0 rows on a replay (already sold) is likewise
// success, not an error. The Kafka acks and the post-commit offer/escrow side-effects
// stay in the consumer.
func (p *ProcessorImpl) SettleMove(req SettleMoveRequest) (SettleMoveResult, error) {
	l := p.l
	ctx := p.ctx
	b := req
	tdb := p.db.WithContext(ctx)
	hid := MoveHoldingId(b.ListingId, b.BuyerId)

	// itemId + sellerId are captured from the listing row inside the tx so the
	// LISTING_SOLD notice emitted on success can carry the sold item id and
	// the seller (so the channel can refresh the seller's panels/wallet).
	// soldSaleType + soldOfferWishSerial drive the offer-purchase side-effects
	// (want-ad consume + sibling-offer release) run after the tx commits.
	var itemId uint32
	var sellerId uint32
	var soldSaleType string
	var soldOfferWishSerial uint32

	err := database.ExecuteTransaction(tdb, func(tx *gorm.DB) error {
		lm, gerr := GetById(b.ListingId.String())(tx)()
		if gerr != nil {
			return gerr
		}
		itemId = lm.TemplateId()
		sellerId = lm.SellerId()
		soldSaleType = string(lm.SaleType())
		soldOfferWishSerial = lm.OfferWishSerial()

		// Conditional ->sold transition. The rows affected is the race
		// arbiter: 1 means this call won the transition; 0 means the listing
		// was already out of its pre-sold state (either this same move already
		// settled it — a replay — or a concurrent cancel/expire won the race).
		//
		// Two valid pre-sold source states feed this step:
		//   - a fixed-price/buy-now Buy settles the listing straight from
		//     `active` (MtsSettlePurchase never pre-transitions the row), so the
		//     buy path is active->sold;
		//   - an auction settle (SettleAuction) pre-transitions the listing
		//     active->settling SYNCHRONOUSLY (the sweep re-discovery guard), so
		//     the auction path is settling->sold.
		// Try active->sold first (the buy path); if 0 rows, try settling->sold
		// (the auction-settle path). Whichever affects 1 row is the winner.
		affected, uerr := UpdateState(tx, b.ListingId.String(), StateActive, StateSold)
		if uerr != nil {
			return uerr
		}
		if affected == 0 {
			affected, uerr = UpdateState(tx, b.ListingId.String(), StateSettling, StateSold)
			if uerr != nil {
				return uerr
			}
		}

		// Idempotency: if the buyer holding already exists for this
		// (listing, buyer), the move has been applied — do not create a
		// second copy.
		if existing, herr := holding.GetById(hid.String())(tx)(); herr == nil && existing.Id() == hid {
			return nil
		}

		// Single-custody guard: when this call did NOT win the active->sold
		// transition (affected==0) and there is no prior buyer holding to
		// idempotently re-ack, the listing was claimed by a concurrent cancel
		// or expire. Creating a buyer holding here would DOUBLE the item (a
		// seller cancel/expire holding plus this purchased holding), so settle
		// must lose the race and create nothing.
		if affected == 0 {
			// Lost the race to a concurrent cancel/expire (no prior holding =
			// not a replay). Creating no holding avoids the double-custody dupe,
			// but the settle MUST fail so the saga compensates the buyer's
			// already-applied prepaid debit. Returning an error emits an ERROR
			// ack -> the move step fails -> reverse-walk re-credits the buyer. A
			// silent success here would charge the buyer for an item the seller
			// reclaimed (currency desync).
			l.Warnf("MtsMoveListingToHolding lost the race for listing [%s] (no active->sold transition, no prior holding); failing settle so the buyer debit is compensated. buyer [%d].", b.ListingId.String(), b.BuyerId)
			return ErrMoveLostRace
		}

		t := tenant.MustFromContext(ctx)
		hm, berr := holding.NewBuilder(t.Id(), world.Id(b.WorldId), b.BuyerId).
			SetId(hid).
			SetOrigin(holding.OriginPurchased).
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
		if _, cerr := holding.CreateHolding(tx, hm); cerr != nil {
			return cerr
		}

		// Record the settle for BOTH parties' My Page -> History. This point
		// is reached only on the winning first settle (the holding-exists
		// guard above returns early on replay), so the two rows are written
		// exactly once. salePrice is the seller's BASE price — the listing
		// value for a fixed/buy-now sale, or the winning bid for an auction.
		// salePrice is taken from the settle payload (buy-now price for a buy-now, list
		// value for a fixed sale, winning bid for an auction settle). Re-deriving it
		// from the listing row recorded the last BID for a buy-now auction instead of
		// the buy-now price (task-102 live finding). A 0 (older in-flight message) falls
		// back to the row-derived value.
		salePrice := b.Price
		if salePrice == 0 {
			salePrice = lm.ListValue()
			if lm.SaleType() == SaleTypeAuction {
				salePrice = lm.CurrentBid()
			}
		}

		// Under the Option B pricing model the buyer pays the commission-
		// inclusive markup while the seller nets the base: the History rows
		// must reflect what each party actually transacted, so the buyer's
		// purchase row records MarkedUp(salePrice) and the seller's sale row
		// records the base salePrice. Recording base on the buyer row made My
		// Page -> History under-report the purchase price (task-102 live finding).
		cfg := configuration.GetRegistry().GetTenantConfig(l, ctx, t.Id())
		buyerPaid := MarkedUp(salePrice, lm.CommissionRate(), cfg.CommissionBase())

		buyerTxn, berr := transaction.NewBuilder(t.Id(), world.Id(b.WorldId), b.BuyerId).
			SetId(uuid.New()).
			SetCounterpartyId(sellerId).
			SetItemId(lm.TemplateId()).
			SetQuantity(lm.Quantity()).
			SetTotalPrice(buyerPaid).
			SetKind(transaction.KindPurchase).
			Build()
		if berr != nil {
			return berr
		}
		if _, terr := transaction.CreateTransaction(tx, buyerTxn); terr != nil {
			return terr
		}

		sellerTxn, berr := transaction.NewBuilder(t.Id(), world.Id(b.WorldId), sellerId).
			SetId(uuid.New()).
			SetCounterpartyId(b.BuyerId).
			SetItemId(lm.TemplateId()).
			SetQuantity(lm.Quantity()).
			SetTotalPrice(salePrice).
			SetKind(transaction.KindSale).
			Build()
		if berr != nil {
			return berr
		}
		if _, terr := transaction.CreateTransaction(tx, sellerTxn); terr != nil {
			return terr
		}
		return nil
	})
	if err != nil {
		return SettleMoveResult{}, err
	}
	return SettleMoveResult{
		HoldingId:           hid,
		ItemId:              itemId,
		SellerId:            sellerId,
		SoldSaleType:        soldSaleType,
		SoldOfferWishSerial: soldOfferWishSerial,
	}, nil
}

// RemoveSpuriousActive hard-deletes a spurious ACTIVE listing by id — the
// late-compensation inverse of Accept. DeleteActive is guarded to state=active, so
// a listing bought/cancelled/settled in the interim is left untouched (0 rows =
// success). It returns the affected count so the consumer can log removed-vs-noop.
func (p *ProcessorImpl) RemoveSpuriousActive(id string) (int64, error) {
	tdb := p.db.WithContext(p.ctx)
	var affected int64
	err := database.ExecuteTransaction(tdb, func(tx *gorm.DB) error {
		n, derr := DeleteActive(tx, id)
		affected = n
		return derr
	})
	return affected, err
}

// RestoreFromHolding reverses a settlement move — the late-compensation inverse of
// SettleMove. In one tx it soft-deletes the deterministic buyer holding
// (MoveHoldingId(listingId, buyerId)) and transitions the listing sold->active, so
// a buy that landed late after the buyer was refunded returns the item to the
// marketplace and leaves the buyer nothing. Both mutations are idempotent (0 rows
// on replay = success).
func (p *ProcessorImpl) RestoreFromHolding(listingId string, buyerId uint32) error {
	tdb := p.db.WithContext(p.ctx)
	hid := MoveHoldingId(uuid.MustParse(listingId), buyerId)

	return database.ExecuteTransaction(tdb, func(tx *gorm.DB) error {
		// Remove the buyer's purchased holding (the delivered item).
		if _, herr := holding.SoftDelete(tx, hid.String()); herr != nil {
			return herr
		}
		// Return the listing to the marketplace (sold -> active). 0 rows on a
		// replay (already active) is success, not an error.
		if _, uerr := UpdateState(tx, listingId, StateSold, StateActive); uerr != nil {
			return uerr
		}
		return nil
	})
}
