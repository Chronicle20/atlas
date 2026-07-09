package listing

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Model is the channel-side view of an atlas-mts marketplace listing, read over
// REST for the browse (GET_ITC_LIST / SEARCH_ITC_LIST) arms. ItcSn is the
// listing's persistent per-(tenant, world) serial (the client's nITCSN) — the
// browse arm emits it as MtsItem.itcSn so the client can address this listing in
// subsequent buy/cancel/bid ITC_OPERATION arms.
type Model struct {
	id            string
	worldId       world.Id
	itcSn         uint32
	sellerId      uint32
	sellerName    string
	saleType      string
	state         string
	templateId    uint32
	quantity      uint32
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
	listValue     uint32
	buyNowPrice   uint32
	contractFee   uint32
	currentBid    uint32
	highBidderId  uint32
	minIncrement  uint32
	bidCount      uint32
	category      string
	subCategory   string
	endsAt        *time.Time
}

func (m Model) Id() string            { return m.id }
func (m Model) WorldId() world.Id     { return m.worldId }
func (m Model) ItcSn() uint32         { return m.itcSn }
func (m Model) SellerId() uint32      { return m.sellerId }
func (m Model) SellerName() string    { return m.sellerName }
func (m Model) SaleType() string      { return m.saleType }
func (m Model) State() string         { return m.state }
func (m Model) TemplateId() uint32    { return m.templateId }
func (m Model) Quantity() uint32      { return m.quantity }
func (m Model) Strength() uint16      { return m.strength }
func (m Model) Dexterity() uint16     { return m.dexterity }
func (m Model) Intelligence() uint16  { return m.intelligence }
func (m Model) Luck() uint16          { return m.luck }
func (m Model) HP() uint16            { return m.hp }
func (m Model) MP() uint16            { return m.mp }
func (m Model) WeaponAttack() uint16  { return m.weaponAttack }
func (m Model) MagicAttack() uint16   { return m.magicAttack }
func (m Model) WeaponDefense() uint16 { return m.weaponDefense }
func (m Model) MagicDefense() uint16  { return m.magicDefense }
func (m Model) Accuracy() uint16      { return m.accuracy }
func (m Model) Avoidability() uint16  { return m.avoidability }
func (m Model) Hands() uint16         { return m.hands }
func (m Model) Speed() uint16         { return m.speed }
func (m Model) Jump() uint16          { return m.jump }
func (m Model) Slots() uint16         { return m.slots }
func (m Model) Level() byte           { return m.level }
func (m Model) ItemLevel() byte       { return m.itemLevel }
func (m Model) ItemExp() uint32       { return m.itemExp }
func (m Model) RingId() uint32        { return m.ringId }
func (m Model) ViciousCount() uint32  { return m.viciousCount }
func (m Model) Flags() uint16         { return m.flags }
func (m Model) ListValue() uint32     { return m.listValue }
func (m Model) BuyNowPrice() uint32   { return m.buyNowPrice }
func (m Model) ContractFee() uint32   { return m.contractFee }
func (m Model) CurrentBid() uint32    { return m.currentBid }
func (m Model) HighBidderId() uint32  { return m.highBidderId }
func (m Model) MinIncrement() uint32  { return m.minIncrement }
func (m Model) BidCount() uint32      { return m.bidCount }
func (m Model) Category() string      { return m.category }
func (m Model) SubCategory() string   { return m.subCategory }
func (m Model) EndsAt() *time.Time    { return m.endsAt }
