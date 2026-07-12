package task

import (
	"context"
	"sync"
	"time"

	"atlas-mts/bid"
	"atlas-mts/listing"
	"atlas-mts/wish"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	// defaultInterval is the sweep cadence when the env var is unset/invalid.
	defaultInterval = 60 * time.Second
	// sweepBatchLimit bounds how many expired listings a single sweep processes.
	// The remainder is logged and picked up on the next tick — the sweep is
	// bounded but never silently truncated (NFR 8.3).
	sweepBatchLimit = 500
)

// PeriodicTask runs the DB-driven auction-expiration sweep at a fixed interval.
// It mirrors the asset-expiration ticker structure (time.Ticker + stopCh +
// sync.WaitGroup, env-driven interval) but is DB-driven rather than
// session-driven: each tick queries the listings table directly for expired
// active auctions across every tenant and applies the local expire transition.
type PeriodicTask struct {
	l        logrus.FieldLogger
	ctx      context.Context
	db       *gorm.DB
	interval time.Duration
	stopCh   chan struct{}
	wg       *sync.WaitGroup
}

// NewPeriodicTask creates the expiration sweep task. A non-positive interval
// falls back to defaultInterval.
func NewPeriodicTask(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, interval time.Duration) *PeriodicTask {
	if interval <= 0 {
		interval = defaultInterval
	}
	return &PeriodicTask{
		l:        l,
		ctx:      ctx,
		db:       db,
		interval: interval,
		stopCh:   make(chan struct{}),
		wg:       &sync.WaitGroup{},
	}
}

// Start launches the ticker loop.
func (t *PeriodicTask) Start() {
	t.wg.Add(1)
	routine.Go(t.l, t.ctx, func(context.Context) { t.run() })
	t.l.Infof("MTS expiration sweep started with interval [%v].", t.interval)
}

// Stop signals the loop to exit and waits for the in-flight tick to finish.
func (t *PeriodicTask) Stop() {
	close(t.stopCh)
	t.wg.Wait()
	t.l.Infoln("MTS expiration sweep stopped.")
}

func (t *PeriodicTask) run() {
	defer t.wg.Done()

	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if _, err := Sweep(t.l, t.ctx, t.db); err != nil {
				t.l.WithError(err).Errorf("MTS expiration sweep failed.")
			}
		case <-t.stopCh:
			return
		}
	}
}

// Sweep performs one DB-driven expiration pass: it discovers active auction
// listings whose ends_at has passed (across ALL tenants) and applies the local
// active->holding(seller) transition with origin=expired to each. It returns the
// number of listings actually expired and logs anything deferred to the next tick.
//
// Tenant context reconstruction (THE crux): the listings table stores only a
// tenant_id uuid — no region/version — so a full tenant.Model cannot be rebuilt
// for tenant.WithContext without fabricating version coordinates. Instead the
// sweep runs cross-tenant: it queries under database.WithoutTenantFilter, and the
// expire transition takes the holding's tenant_id from the listing ROW itself
// (lm.TenantId()), so no tenant model reconstruction is needed. Each listing is
// addressed by its unique surrogate uuid, so the per-listing GetById/UpdateState
// inside Expire resolve the correct row without tenant filtering.
func Sweep(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, opts ...listing.Option) (int, error) {
	now := time.Now()
	sweepCtx := database.WithoutTenantFilter(ctx)
	sdb := db.WithContext(sweepCtx)

	total, err := listing.CountExpiredActive(now)(sdb)
	if err != nil {
		return 0, err
	}
	if total == 0 {
		l.Debugln("MTS expiration sweep: no expired auction listings.")
		return 0, nil
	}

	expired, err := listing.GetExpiredActive(now, sweepBatchLimit)(sdb)()
	if err != nil {
		return 0, err
	}

	// The processor shares Cancel's atomic tx; the WithoutTenantFilter context lets
	// it address rows by their unique id across tenants, and the holding tenant is
	// derived from each listing row.
	p := listing.NewProcessor(l, sweepCtx, db, opts...)

	swept := 0
	for _, lm := range expired {
		// Settle-at-expiry decision (design §5.6): an expired auction WITH a high
		// bidder is SETTLED to the winner (seller points credit + custody move,
		// NO winner re-debit — the winner's prepaid was already escrowed at bid
		// time); an expired listing with NO bids returns to the SELLER holding via
		// the Expire transition (origin=expired). SettleAuction encapsulates both
		// arms; the ticker only supplies the resolved winner/seller accounts.
		if lm.HighBidderId() != 0 {
			winnerAccount := winnerAccountFor(sdb, lm.Id(), lm.HighBidderId())
			res, serr := p.SettleAuction(listing.SettleRequest{
				ListingId:       lm.Id(),
				WorldId:         world.Id(lm.WorldId()),
				WinnerId:        lm.HighBidderId(),
				WinnerAccountId: winnerAccount,
				SellerAccountId: lm.SellerAccountId(),
			})
			if serr != nil {
				l.WithError(serr).Warnf("MTS expiration sweep: failed to settle auction [%s] to winner [%d] (tenant [%s]); will retry next tick.", lm.Id(), lm.HighBidderId(), lm.TenantId())
				continue
			}
			if res.HadWinner {
				swept++
				l.Debugf("MTS expiration sweep: settled auction [%s] -> winner [%d] holding (tenant [%s]).", lm.Id(), lm.HighBidderId(), lm.TenantId())
			} else if res.Expired {
				// The high bidder had no held bid (e.g. a stale/released row); the
				// auction returned to the seller holding instead.
				swept++
				l.Debugf("MTS expiration sweep: auction [%s] had a high bidder but no held bid; returned to seller [%d] holding.", lm.Id(), lm.SellerId())
			}
			continue
		}

		res, eerr := p.Expire(lm.Id().String())
		if eerr != nil {
			l.WithError(eerr).Warnf("MTS expiration sweep: failed to expire listing [%s] (tenant [%s], seller [%d]); will retry next tick.", lm.Id(), lm.TenantId(), lm.SellerId())
			continue
		}
		if res.Won {
			swept++
			l.Debugf("MTS expiration sweep: expired listing [%s] -> seller [%d] holding (tenant [%s]).", lm.Id(), res.SellerId, lm.TenantId())
		}
		// res.Won==false means a concurrent buy already settled the row between the
		// discovery query and the transition; that is correct (the buyer won) and
		// is not counted as an expiration.
	}

	deferred := int(total) - len(expired)
	if deferred > 0 {
		l.Infof("MTS expiration sweep: expired/settled [%d] listings this tick; [%d] remain past the [%d] batch cap and will be processed next tick.", swept, deferred, sweepBatchLimit)
	} else {
		l.Infof("MTS expiration sweep: expired/settled [%d] of [%d] discovered listings.", swept, len(expired))
	}

	// Delete expired "wanted" want-ads across every tenant (cart entries carry no
	// expiry and are never touched). This is a best-effort tail of the same sweep:
	// a delete failure is logged and does NOT fail the listing sweep, which has
	// already committed its transitions above.
	if deleted, derr := wish.DeleteExpiredWanted(sdb, now); derr != nil {
		l.WithError(derr).Warnf("MTS expiration sweep: failed to delete expired want-ads; will retry next tick.")
	} else if deleted > 0 {
		l.Infof("MTS expiration sweep: deleted [%d] expired want-ad(s).", deleted)
	}

	return swept, nil
}

// winnerAccountFor resolves the auction winner's cash-shop account id from THEIR
// winning held bid row (the bid carries the bidder account captured at bid time).
// It returns 0 if no held bid is found for the winner, in which case SettleAuction
// takes the no-held-bid -> seller-holding fallback. The sweep handle is already
// WithoutTenantFilter-scoped, so the row is addressed by its listing id across
// tenants.
func winnerAccountFor(db *gorm.DB, listingId uuid.UUID, winnerId uint32) uint32 {
	bids, err := bid.GetByListingId(listingId)(db)()
	if err != nil {
		return 0
	}
	for _, b := range bids {
		if b.BidderId() == winnerId && b.State() == bid.StateHeld {
			return b.BidderAccountId()
		}
	}
	return 0
}
