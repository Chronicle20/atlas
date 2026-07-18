package listing

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// SaleType enumerates how a listing is sold.
type SaleType string

const (
	SaleTypeFixed   SaleType = "fixed"
	SaleTypeAuction SaleType = "auction"
	// SaleTypeOffer is an item escrowed against a want-ad, sold only to that want-ad's poster.
	SaleTypeOffer SaleType = "offer"
)

// State enumerates the lifecycle state of a listing.
type State string

const (
	StateActive State = "active"
	// StateSettling is the transient state an auction listing enters SYNCHRONOUSLY
	// the moment its settle-to-winner saga is emitted (active->settling), before the
	// async MtsMoveListingToHolding custody step flips it settling->sold. It exists
	// solely to take a settling auction OUT of the expiration sweep's discovery set
	// (state='active' AND ends_at<now) so a second sweep tick cannot re-discover the
	// row and emit a SECOND seller-credit saga (the double-credit money bug). The
	// active->settling transition is a race-safe CAS, so two concurrent ticks cannot
	// both emit. On a settle-emit failure the row is reverted settling->active.
	StateSettling  State = "settling"
	StateSold      State = "sold"
	StateCancelled State = "cancelled"
	StateExpired   State = "expired"
)

// Model is the immutable per-world marketplace listing. It carries an explicit
// item snapshot (template id, quantity, and the full equip stat block) plus
// sale/auction/state fields. Construct it via the Builder.
type Model struct {
	id              uuid.UUID
	tenantId        uuid.UUID
	worldId         world.Id
	serial          uint32
	sellerId        uint32
	sellerAccountId uint32
	sellerName      string

	saleType SaleType
	state    State

	// item snapshot
	templateId uint32
	quantity   uint32

	// equip stat block
	strength      uint16
	dexterity     uint16
	intelligence  uint16
	luck          uint16
	hp            uint16
	mp            uint16
	weaponAttack  uint16
	magicAttack   uint16
	weaponDefense uint16
	magicDefense  uint16
	accuracy      uint16
	avoidability  uint16
	hands         uint16
	speed         uint16
	jump          uint16
	slots         uint16
	level         byte
	itemLevel     byte
	itemExp       uint32
	ringId        uint32
	viciousCount  uint32
	flags         uint16
	owner         string

	// sale fields
	listValue      uint32
	buyNowPrice    *uint32
	commissionRate float64
	category       string
	subCategory    string

	// offer linkage: an offer listing points at the want-ad it fulfills
	// (serial + poster id); 0 for normal listings.
	offerWishSerial  uint32
	offerWishOwnerId uint32

	// auction fields
	endsAt       *time.Time
	currentBid   uint32
	highBidderId uint32
	minIncrement uint32
	bidCount     uint32

	createdAt time.Time
	updatedAt time.Time
}

func (m Model) Id() uuid.UUID           { return m.id }
func (m Model) TenantId() uuid.UUID     { return m.tenantId }
func (m Model) WorldId() world.Id       { return m.worldId }
func (m Model) Serial() uint32          { return m.serial }
func (m Model) SellerId() uint32        { return m.sellerId }
func (m Model) SellerAccountId() uint32 { return m.sellerAccountId }
func (m Model) SellerName() string      { return m.sellerName }
func (m Model) SaleType() SaleType      { return m.saleType }
func (m Model) State() State            { return m.state }
func (m Model) TemplateId() uint32      { return m.templateId }
func (m Model) Quantity() uint32        { return m.quantity }
func (m Model) Strength() uint16        { return m.strength }
func (m Model) Dexterity() uint16       { return m.dexterity }
func (m Model) Intelligence() uint16    { return m.intelligence }
func (m Model) Luck() uint16            { return m.luck }
func (m Model) HP() uint16              { return m.hp }
func (m Model) MP() uint16              { return m.mp }
func (m Model) WeaponAttack() uint16    { return m.weaponAttack }
func (m Model) MagicAttack() uint16     { return m.magicAttack }
func (m Model) WeaponDefense() uint16   { return m.weaponDefense }
func (m Model) MagicDefense() uint16    { return m.magicDefense }
func (m Model) Accuracy() uint16        { return m.accuracy }
func (m Model) Avoidability() uint16    { return m.avoidability }
func (m Model) Hands() uint16           { return m.hands }
func (m Model) Speed() uint16           { return m.speed }
func (m Model) Jump() uint16            { return m.jump }
func (m Model) Slots() uint16           { return m.slots }
func (m Model) Level() byte             { return m.level }
func (m Model) ItemLevel() byte         { return m.itemLevel }
func (m Model) ItemExp() uint32         { return m.itemExp }
func (m Model) RingId() uint32          { return m.ringId }
func (m Model) ViciousCount() uint32    { return m.viciousCount }
func (m Model) Flags() uint16           { return m.flags }
func (m Model) Owner() string           { return m.owner }

func (m Model) ListValue() uint32       { return m.listValue }
func (m Model) BuyNowPrice() *uint32    { return m.buyNowPrice }
func (m Model) CommissionRate() float64 { return m.commissionRate }
func (m Model) Category() string        { return m.category }
func (m Model) SubCategory() string     { return m.subCategory }

func (m Model) OfferWishSerial() uint32  { return m.offerWishSerial }
func (m Model) OfferWishOwnerId() uint32 { return m.offerWishOwnerId }

func (m Model) EndsAt() *time.Time   { return m.endsAt }
func (m Model) CurrentBid() uint32   { return m.currentBid }
func (m Model) HighBidderId() uint32 { return m.highBidderId }
func (m Model) MinIncrement() uint32 { return m.minIncrement }
func (m Model) BidCount() uint32     { return m.bidCount }

func (m Model) CreatedAt() time.Time { return m.createdAt }
func (m Model) UpdatedAt() time.Time { return m.updatedAt }
