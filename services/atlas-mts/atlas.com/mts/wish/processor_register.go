package wish

import (
	"atlas-mts/configuration"
	"atlas-mts/listing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// RegisterWishRequest carries the wish-registration parameters the RegisterWish
// command applies. WishType is resolved by the consumer from the command Origin
// (the wire concern) so the wish package needs no dependency on the kafka message
// package; the price/expiry business logic (want-ad base + fixed-sale expiry) lives
// in the processor method.
type RegisterWishRequest struct {
	WishId        uuid.UUID
	WorldId       world.Id
	CharacterId   uint32
	ItemId        uint32
	WishType      string
	ListingSerial uint32
	Count         uint32
	Price         uint32
}

// RegisterWish creates a wish-list entry for a character using the caller-supplied
// WishId. For a "wanted" want-ad it derives the offerer's BASE price
// (listing.WantAdBaseFromTotal) from the poster's commission-INCLUSIVE typed total
// and sets the fixed-sale expiry; a "cart" entry stores the price as-is and never
// expires. The create runs in one local DB transaction. The WISH_ADDED emission
// stays in the consumer.
func (p *ProcessorImpl) RegisterWish(req RegisterWishRequest) error {
	tdb := p.db.WithContext(p.ctx)

	return database.ExecuteTransaction(tdb, func(tx *gorm.DB) error {
		t := tenant.MustFromContext(p.ctx)
		wb := NewBuilder(t.Id(), req.CharacterId, req.ItemId).
			SetId(req.WishId).
			SetWorldId(req.WorldId).
			SetType(req.WishType).
			SetListingSerial(req.ListingSerial).
			SetCount(req.Count)
		// A "wanted" want-ad's price is the poster's commission-INCLUSIVE total
		// (the register dialog sends the raw typed amount — CRegisterWishEntryDlg::
		// Confirm does no commission math). Store the BASE the offerer nets
		// (UnMarkUp) so an offer credits base and the poster pays MarkedUp(base) ==
		// the total; the commission is the platform's sink, like a normal buy. A
		// "cart" entry's price is the favorited listing's list value (already base),
		// stored as-is. A wanted entry also expires after the tenant fixed-sale term
		// (the periodic sweep hard-deletes it); a cart entry never expires.
		price := req.Price
		if req.WishType == TypeWanted {
			cfg := configuration.GetRegistry().GetTenantConfig(p.l, p.ctx, t.Id())
			price = listing.WantAdBaseFromTotal(req.Price, cfg.CommissionRate(), cfg.CommissionBase())
			exp := time.Now().Add(time.Duration(cfg.FixedSaleDurationHours()) * time.Hour)
			wb = wb.SetExpiresAt(&exp)
		}
		wb = wb.SetPrice(price)
		wm, berr := wb.Build()
		if berr != nil {
			return berr
		}
		_, cerr := CreateWish(tx, wm)
		return cerr
	})
}

// RemoveWish deletes a wish-list entry by id in one local DB transaction. The row
// is read inside the tx before the delete so the returned characterId can echo the
// owning character onto the WISH_REMOVED event; a missing row (already removed)
// leaves characterId 0 and the delete affects 0 rows — both are success, not
// errors. The WISH_REMOVED emission stays in the consumer.
func (p *ProcessorImpl) RemoveWish(id string) (uint32, error) {
	tdb := p.db.WithContext(p.ctx)

	var characterId uint32
	err := database.ExecuteTransaction(tdb, func(tx *gorm.DB) error {
		// Read the row first so the event can echo the owning characterId.
		// A missing row (already removed) leaves characterId 0 and the
		// delete affects 0 rows — both are success, not errors.
		if wm, gerr := GetById(id)(tx)(); gerr == nil {
			characterId = wm.CharacterId()
		}
		_, derr := DeleteWish(tx, id)
		return derr
	})
	if err != nil {
		return 0, err
	}
	return characterId, nil
}
