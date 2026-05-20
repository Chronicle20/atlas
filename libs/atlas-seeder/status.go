package seeder

import (
	"context"
	"sync"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

func ReadStatus(ctx context.Context, db *gorm.DB, src CatalogSource, g Group) (Status, error) {
	t := tenant.MustFromContext(ctx)
	out := Status{
		GroupName:  g.Name,
		Subdomains: make(map[string]SubdomainStatus, len(g.Subdomains)),
	}

	roots, err := src.Roots(t)
	if err == nil && len(roots) > 0 {
		rev, _ := src.Revision(roots[0])
		out.CatalogRevision = rev
	}

	row, err := ReadSeedState(db.WithContext(ctx), t.Id(), g.Name)
	if err != nil {
		return out, err
	}
	if row != nil {
		rev := row.CatalogRevision
		out.TenantSeededRevision = &rev
		ts := row.SeededAt
		out.TenantSeededAt = &ts
	}

	var mu sync.Mutex
	var latest *time.Time
	eg, gctx := errgroup.WithContext(ctx)
	for _, sd := range g.Subdomains {
		sd := sd
		eg.Go(func() error {
			count, ts, err := sd.Count(db.WithContext(gctx))
			if err != nil {
				return err
			}
			mu.Lock()
			out.Subdomains[sd.Name()] = SubdomainStatus{Count: count, UpdatedAt: ts}
			if ts != nil && (latest == nil || ts.After(*latest)) {
				latest = ts
			}
			mu.Unlock()
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return out, err
	}
	out.UpdatedAt = latest
	return out, nil
}
