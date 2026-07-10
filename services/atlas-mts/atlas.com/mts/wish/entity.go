package wish

import (
	"time"

	"atlas-mts/serial"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Migration creates the wish_entries table (and the shared mts_serials counter
// it draws from). It is a brand-new table (no legacy primary-key rewrite), so
// AutoMigrate alone produces the correct surrogate-key shape and the composite
// indexes declared on the entity tags.
func Migration(db *gorm.DB) error {
	if err := serial.Migration(db); err != nil {
		return err
	}
	// The (tenant, world, character, item) unique index was widened to include
	// `type` (cart vs wanted) so a character can hold the same item as both a cart
	// entry and a wanted entry. AutoMigrate will not alter an existing index in
	// place, so drop the old one first and let AutoMigrate recreate it with the new
	// column set.
	if db.Migrator().HasIndex(&entity{}, "idx_wish_entries_char_item") {
		_ = db.Migrator().DropIndex(&entity{}, "idx_wish_entries_char_item")
	}
	return db.AutoMigrate(&entity{})
}

// entity is the GORM row for a wish-list entry.
//
// The primary key is a surrogate UUID (Id); business identity is never the key,
// and a (tenant_id, id) unique index keeps the row tenant-scoped — never a
// unique index on tenant_id alone, which would cap a tenant at one wish entry.
//
// Serial is the per-(tenant, world) ITC serial (the client's nITCSN) drawn from
// the shared `serial` counter at create time; a (tenant_id, world_id, serial)
// unique index lets GetBySerial resolve a CANCEL_WISH serial back to the wish
// entry. tenant_id is part of the unique key because the serial counter is
// per-(tenant, world): serial 1 recurs across tenants and across worlds.
//
// Composite indexes:
//   - (tenant_id, character_id)                       — a character's wish list
//   - (tenant_id, world_id, serial) UNIQUE            — serial -> wish entry
//   - (tenant_id, world_id, character_id, item_id) UQ — one wish per (char, item)
//     per world, enforcing the design's "one wish per (character, item)" invariant
//     and making the idempotent create well-defined.
type entity struct {
	Id          uuid.UUID `gorm:"column:id;type:uuid;primaryKey;uniqueIndex:idx_wish_entries_tenant_id,priority:2"`
	TenantId    uuid.UUID `gorm:"column:tenant_id;type:uuid;not null;uniqueIndex:idx_wish_entries_tenant_id,priority:1;index:idx_wish_entries_character,priority:1;uniqueIndex:idx_wish_entries_world_serial,priority:1;uniqueIndex:idx_wish_entries_char_item,priority:1"`
	WorldId     byte      `gorm:"column:world_id;not null;uniqueIndex:idx_wish_entries_world_serial,priority:2;uniqueIndex:idx_wish_entries_char_item,priority:2"`
	Serial      uint32    `gorm:"column:serial;not null;uniqueIndex:idx_wish_entries_world_serial,priority:3"`
	CharacterId uint32    `gorm:"column:character_id;not null;index:idx_wish_entries_character,priority:2;uniqueIndex:idx_wish_entries_char_item,priority:3"`
	ItemId      uint32    `gorm:"column:item_id;not null;uniqueIndex:idx_wish_entries_char_item,priority:4"`
	// ListingSerial is the ITC serial of the specific LISTING a "cart" (SET_ZZIM)
	// entry favorited. The Cart renders and settles THAT exact listing (GetBySerial)
	// instead of re-resolving the item template to some other seller's cheapest
	// listing — the old behavior showed a different listing's price and sold-until
	// date and could not be bought (task-102 live finding). 0 for "wanted" entries,
	// which reference no listing. A plain new defaulted column — AutoMigrate adds it
	// with no index/key changes.
	ListingSerial uint32 `gorm:"column:listing_serial;not null;default:0"`
	// Type distinguishes a "cart" entry (added-to-cart, SET_ZZIM) from a "wanted"
	// entry (a want-ad, REGISTER_WISH_ENTRY); part of the char_item unique index so
	// the same item can be in both the cart and the wanted list.
	Type string `gorm:"column:type;not null;default:cart;uniqueIndex:idx_wish_entries_char_item,priority:5"`
	// Price is the wish entry's price: for a "wanted" entry it is the want-ad price
	// the registrant offered (REGISTER_WISH_ENTRY); for a "cart" entry it is the
	// favorited listing's list value at the time it was carted (SET_ZZIM). It backs
	// the price the Cart / Wanted views render. A plain new defaulted column —
	// AutoMigrate adds it with no index/key changes.
	Price uint32 `gorm:"column:price;not null;default:0"`
	// Count is the requested quantity for a "wanted" entry (REGISTER_WISH_ENTRY):
	// how many of the item the want-ad is asking for. An offer against the want-ad
	// escrows only this many units (not the offerer's full stack). Floored to 1 at
	// create time. A plain new defaulted column — AutoMigrate adds it.
	Count uint32 `gorm:"column:count;not null;default:1"`
	// ExpiresAt is the "wanted" want-ad's expiry (created_at + the tenant fixed-sale
	// term). NULL for "cart" entries, which never expire. Nullable so a cart entry
	// carries no expiry and the sweep only ever deletes wanted rows with a set
	// expiry that has passed.
	ExpiresAt *time.Time `gorm:"column:expires_at"`

	CreatedAt time.Time `gorm:"column:created_at"`
}

func (e entity) TableName() string {
	return "wish_entries"
}
