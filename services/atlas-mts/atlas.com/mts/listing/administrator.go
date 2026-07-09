package listing

import (
	"fmt"
	"time"

	"atlas-mts/serial"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// parseId converts a string id into a uuid, returning uuid.Nil on a malformed
// value so a bad path param degrades to a not-found query rather than panicking.
func parseId(id string) uuid.UUID {
	u, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil
	}
	return u
}

// GetById is the exported provider wrapper: it resolves a listing by surrogate
// id, mapping the entity to the immutable Model.
func GetById(id string) database.EntityProvider[Model] {
	return func(db *gorm.DB) model.Provider[Model] {
		return model.Map(modelFromEntity)(getById(id)(db))
	}
}

// GetBySerial is the exported provider wrapper: it resolves a listing by its
// per-(tenant, world) ITC serial (the client's nITCSN), mapping the entity to the
// immutable Model.
func GetBySerial(worldId world.Id, sn uint32) database.EntityProvider[Model] {
	return func(db *gorm.DB) model.Provider[Model] {
		return model.Map(modelFromEntity)(getBySerial(worldId, sn)(db))
	}
}

// GetAll resolves every listing visible to the request's tenant.
func GetAll() database.EntityProvider[[]Model] {
	return func(db *gorm.DB) model.Provider[[]Model] {
		return model.SliceMap(modelFromEntity)(getAll()(db))()
	}
}

// GetExpiredActive resolves the active auction listings whose ends_at has passed
// (not null, < now), capped at limit (0 = uncapped). Tenant scoping is the
// caller's responsibility via the db context: the expiration sweep passes a
// WithoutTenantFilter context to discover expired listings across every tenant.
func GetExpiredActive(now time.Time, limit int) database.EntityProvider[[]Model] {
	return func(db *gorm.DB) model.Provider[[]Model] {
		return model.SliceMap(modelFromEntity)(getExpiredActive(now, limit)(db))()
	}
}

// CountExpiredActive returns the total number of expired active auction listings
// (ignoring any batch limit), so the sweep can log how many it deferred to the
// next tick rather than silently truncating (NFR 8.3).
func CountExpiredActive(now time.Time) func(db *gorm.DB) (int64, error) {
	return countExpiredActive(now)
}

// CreateListing assigns a fresh surrogate id, draws the next per-(tenant, world)
// ITC serial (the client's nITCSN, shared with holdings), persists an
// explicit-column row, and returns the stored Model.
//
// The serial is drawn here — at the INSERT choke point — using the SAME db handle
// the caller passes. Every production caller (the AcceptToMtsListing custody
// handler) invokes CreateListing only AFTER an id-existence check inside an
// ExecuteTransaction, so a replayed create short-circuits before reaching here and
// never consumes a serial; the serial draw and the row insert then commit or roll
// back together within that one transaction.
func CreateListing(db *gorm.DB, m Model) (Model, error) {
	id := m.Id()
	if id == uuid.Nil {
		id = uuid.New()
	}
	now := time.Now()
	createdAt := m.CreatedAt()
	if createdAt.IsZero() {
		createdAt = now
	}

	sn, err := serial.Next(db, m.TenantId(), m.WorldId())
	if err != nil {
		return Model{}, err
	}

	e := entity{
		Id:              id,
		TenantId:        m.TenantId(),
		WorldId:         byte(m.WorldId()),
		Serial:          sn,
		SellerId:        m.SellerId(),
		SellerAccountId: m.SellerAccountId(),
		SellerName:      m.SellerName(),
		SaleType:        string(m.SaleType()),
		State:           string(m.State()),
		TemplateId:      m.TemplateId(),
		Quantity:        m.Quantity(),
		Strength:        m.Strength(),
		Dexterity:       m.Dexterity(),
		Intelligence:    m.Intelligence(),
		Luck:            m.Luck(),
		HP:              m.HP(),
		MP:              m.MP(),
		WeaponAttack:    m.WeaponAttack(),
		MagicAttack:     m.MagicAttack(),
		WeaponDefense:   m.WeaponDefense(),
		MagicDefense:    m.MagicDefense(),
		Accuracy:        m.Accuracy(),
		Avoidability:    m.Avoidability(),
		Hands:           m.Hands(),
		Speed:           m.Speed(),
		Jump:            m.Jump(),
		Slots:           m.Slots(),
		Level:           m.Level(),
		ItemLevel:       m.ItemLevel(),
		ItemExp:         m.ItemExp(),
		RingId:          m.RingId(),
		ViciousCount:    m.ViciousCount(),
		Flags:           m.Flags(),
		ListValue:       m.ListValue(),
		BuyNowPrice:     m.BuyNowPrice(),
		CommissionRate:  m.CommissionRate(),
		Category:        m.Category(),
		SubCategory:     m.SubCategory(),
		EndsAt:          m.EndsAt(),
		CurrentBid:      m.CurrentBid(),
		HighBidderId:    m.HighBidderId(),
		MinIncrement:    m.MinIncrement(),
		CreatedAt:       createdAt,
		UpdatedAt:       now,
	}
	if err := db.Create(&e).Error; err != nil {
		return Model{}, err
	}
	return modelFromEntity(e)
}

// UpdateState performs the race-safe conditional transition. It updates the row
// only when its current state equals `from`, returning the number of rows
// affected: 1 on a successful transition, 0 if another writer already moved the
// row out of `from` (the cancel-vs-buy race). The tenant callback scopes the
// write to the request's tenant.
func UpdateState(db *gorm.DB, id string, from State, to State) (int64, error) {
	lid := parseId(id)
	if lid == uuid.Nil {
		// Guard against the GORM zero-value struct-condition elision: a uuid.Nil
		// Id condition would vanish, degrading the update to a tenant-wide
		// transition of every listing still in `from`.
		return 0, fmt.Errorf("invalid listing id %q", id)
	}
	// The map-keyed WHERE forces the id into the query; the conditional
	// state = from predicate is preserved for the race-safe transition.
	result := db.Model(&entity{}).
		Where(map[string]interface{}{"id": lid, "state": string(from)}).
		Updates(map[string]interface{}{
			"state":      string(to),
			"updated_at": time.Now(),
		})
	return result.RowsAffected, result.Error
}

// DeleteActive hard-deletes a listing row by id ONLY while it is still active,
// returning rows affected (1 = deleted, 0 = not active / already gone). It is the
// late-compensation inverse of a spurious AcceptToMtsListing: the list saga timed
// out, its compensation re-granted the item to the seller, and this removes the
// duplicate listing the late accept created. The state=active guard is the safety
// key — a listing bought/cancelled/settled in the interim is left untouched
// (0 rows, success), never destroying a legitimate sold/holding-backed row.
func DeleteActive(db *gorm.DB, id string) (int64, error) {
	lid := parseId(id)
	if lid == uuid.Nil {
		// Guard against the GORM zero-value struct-condition elision: a uuid.Nil
		// Id condition would vanish, degrading the delete to a tenant-wide wipe of
		// every active listing.
		return 0, fmt.Errorf("invalid listing id %q", id)
	}
	result := db.Where(map[string]interface{}{"id": lid, "state": string(StateActive)}).Delete(&entity{})
	return result.RowsAffected, result.Error
}

// UpdateAuction updates the live auction fields (current bid, high bidder, and
// the optional end time) within a transaction. Used by the bid path.
func UpdateAuction(db *gorm.DB, id string, currentBid uint32, highBidderId uint32, endsAt *time.Time) error {
	lid := parseId(id)
	if lid == uuid.Nil {
		// Guard against the GORM zero-value struct-condition elision. This is the
		// most dangerous write: UpdateAuction has NO state predicate, so an elided
		// id would rewrite EVERY listing's auction fields tenant-wide.
		return fmt.Errorf("invalid listing id %q", id)
	}
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		return tx.Model(&entity{}).
			Where(map[string]interface{}{"id": lid}).
			Updates(map[string]interface{}{
				"current_bid":    currentBid,
				"high_bidder_id": highBidderId,
				"ends_at":        endsAt,
				"updated_at":     time.Now(),
			}).Error
	})
}

// AdvanceAuctionBid is the race-safe compare-and-swap that advances an active
// auction's high bid. It updates current_bid/high_bidder_id only when the row is
// still active AND its current high bid matches the prior values the caller read
// (the optimistic-concurrency arbiter): a concurrent bid that already advanced the
// row makes this caller the loser (0 rows affected). It returns the number of rows
// affected — 1 on a win, 0 on a lost race. endsAt is intentionally NOT touched
// here (no anti-snipe extension in this task); the auction's ends_at stays fixed.
//
// The map-keyed WHERE forces every condition into the query (a struct condition
// would elide a zero-valued prior bid/bidder, which is exactly the first-bid case).
func AdvanceAuctionBid(db *gorm.DB, id string, priorBid uint32, priorBidder uint32, newBid uint32, newBidder uint32) (int64, error) {
	result := db.Model(&entity{}).
		Where(map[string]interface{}{
			"id":             parseId(id),
			"state":          string(StateActive),
			"current_bid":    priorBid,
			"high_bidder_id": priorBidder,
		}).
		Updates(map[string]interface{}{
			"current_bid":    newBid,
			"high_bidder_id": newBidder,
			"bid_count":      gorm.Expr("bid_count + 1"),
			"updated_at":     time.Now(),
		})
	return result.RowsAffected, result.Error
}

// BackdateEndsAt rewrites ends_at on an ACTIVE listing that HAS a sale term —
// the test-route time-travel primitive (design-e2e-testing.md §4.2). Both
// auctions and fixed sales carry terms (era-faithful fixed-sale expiry); only
// legacy rows with a NULL ends_at are refused. The guards live in the WHERE
// clause so the update is race-safe: a listing settled between the caller's
// read and this write is left untouched (0 rows affected), exactly like
// UpdateState. Everything downstream of the rewritten timestamp (sweep
// discovery, settle/expire arms) is production code.
func BackdateEndsAt(db *gorm.DB, id string, to time.Time) (int64, error) {
	res := db.Model(&entity{}).
		Where("id = ? AND state = ? AND ends_at IS NOT NULL", parseId(id), string(StateActive)).
		Update("ends_at", to)
	return res.RowsAffected, res.Error
}
