package listing

import (
	"time"

	"atlas-mts/serial"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Migration creates the listings table. It is a brand-new table (no legacy
// primary-key rewrite), so AutoMigrate alone produces the correct surrogate-key
// shape and the composite indexes declared on the entity tags. It also migrates
// the shared per-(tenant, world) ITC-serial counter table, since CreateListing
// draws a serial from it on every insert — co-migrating keeps the dependency
// satisfied for every caller (prod boot and the per-package test harness alike).
func Migration(db *gorm.DB) error {
	if err := serial.Migration(db); err != nil {
		return err
	}
	return db.AutoMigrate(&entity{})
}

// entity is the GORM row for a marketplace listing.
//
// The primary key is a surrogate UUID (Id); business identity is never the key,
// and a (tenant_id, id) unique index keeps the row tenant-scoped. The item
// snapshot is stored as explicit name-keyed columns — one column per stat, no
// JSON blob — so a binary COPY/restore is column-order safe.
//
// Composite indexes back the design's hot queries:
//   - (tenant_id, world_id, state, category) — browse a world's active listings by category
//   - (tenant_id, seller_id, state)          — a seller's own listings
//   - (tenant_id, world_id, ends_at)         — auction-expiry sweep
//   - (tenant_id, world_id, serial) UNIQUE   — serial->row resolution for the
//     ITC_OPERATION arms; the serial is the client's nITCSN, assigned from the
//     shared per-(tenant, world) counter at row creation. UNIQUE so a serial maps
//     to exactly one row within a world (and never collides with a holding, since
//     listings and holdings draw from the same counter).
type entity struct {
	Id              uuid.UUID `gorm:"column:id;type:uuid;primaryKey;uniqueIndex:idx_listings_tenant_id,priority:2"`
	TenantId        uuid.UUID `gorm:"column:tenant_id;type:uuid;not null;uniqueIndex:idx_listings_tenant_id,priority:1;index:idx_listings_world_state_category,priority:1;index:idx_listings_seller_state,priority:1;index:idx_listings_world_ends_at,priority:1;uniqueIndex:idx_listings_world_serial,priority:1"`
	WorldId         byte      `gorm:"column:world_id;not null;index:idx_listings_world_state_category,priority:2;index:idx_listings_world_ends_at,priority:2;uniqueIndex:idx_listings_world_serial,priority:2"`
	Serial          uint32    `gorm:"column:serial;not null;uniqueIndex:idx_listings_world_serial,priority:3"`
	SellerId        uint32    `gorm:"column:seller_id;not null;index:idx_listings_seller_state,priority:2"`
	SellerAccountId uint32    `gorm:"column:seller_account_id;not null"`
	SellerName      string    `gorm:"column:seller_name;not null"`

	SaleType string `gorm:"column:sale_type;not null"`
	State    string `gorm:"column:state;not null;index:idx_listings_world_state_category,priority:3;index:idx_listings_seller_state,priority:3"`

	TemplateId uint32 `gorm:"column:template_id;not null"`
	Quantity   uint32 `gorm:"column:quantity;not null"`

	Strength      uint16 `gorm:"column:strength;not null"`
	Dexterity     uint16 `gorm:"column:dexterity;not null"`
	Intelligence  uint16 `gorm:"column:intelligence;not null"`
	Luck          uint16 `gorm:"column:luck;not null"`
	HP            uint16 `gorm:"column:hp;not null"`
	MP            uint16 `gorm:"column:mp;not null"`
	WeaponAttack  uint16 `gorm:"column:weapon_attack;not null"`
	MagicAttack   uint16 `gorm:"column:magic_attack;not null"`
	WeaponDefense uint16 `gorm:"column:weapon_defense;not null"`
	MagicDefense  uint16 `gorm:"column:magic_defense;not null"`
	Accuracy      uint16 `gorm:"column:accuracy;not null"`
	Avoidability  uint16 `gorm:"column:avoidability;not null"`
	Hands         uint16 `gorm:"column:hands;not null"`
	Speed         uint16 `gorm:"column:speed;not null"`
	Jump          uint16 `gorm:"column:jump;not null"`
	Slots         uint16 `gorm:"column:slots;not null"`
	Level         byte   `gorm:"column:level;not null"`
	ItemLevel     byte   `gorm:"column:item_level;not null"`
	ItemExp       uint32 `gorm:"column:item_exp;not null"`
	RingId        uint32 `gorm:"column:ring_id;not null"`
	ViciousCount  uint32 `gorm:"column:vicious_count;not null"`
	Flags         uint16 `gorm:"column:flags;not null"`

	ListValue      uint32  `gorm:"column:list_value;not null"`
	BuyNowPrice    *uint32 `gorm:"column:buy_now_price"`
	CommissionRate float64 `gorm:"column:commission_rate;not null"`
	Category       string  `gorm:"column:category;not null"`
	SubCategory    string  `gorm:"column:sub_category;not null"`

	// OfferWishSerial / OfferWishOwnerId link an `offer` listing to the want-ad
	// it fulfills (serial + poster id); 0 for normal listings. AutoMigrate adds
	// them (no index changes).
	OfferWishSerial  uint32 `gorm:"column:offer_wish_serial;not null;default:0"`
	OfferWishOwnerId uint32 `gorm:"column:offer_wish_owner_id;not null;default:0"`

	EndsAt       *time.Time `gorm:"column:ends_at;index:idx_listings_world_ends_at,priority:3"`
	CurrentBid   uint32     `gorm:"column:current_bid;not null"`
	HighBidderId uint32     `gorm:"column:high_bidder_id;not null"`
	MinIncrement uint32     `gorm:"column:min_increment;not null"`
	BidCount     uint32     `gorm:"column:bid_count;not null;default:0"`

	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (e entity) TableName() string {
	return "listings"
}
