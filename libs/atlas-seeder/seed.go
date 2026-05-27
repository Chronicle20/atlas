package seeder

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"regexp"
	"sync"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Seed(ctx context.Context, db *gorm.DB, src CatalogSource, g Group) (Result, error) {
	t := tenant.MustFromContext(ctx)
	started := time.Now().UTC()

	roots, err := src.Roots(t)
	if err != nil {
		return persistAndReturn(ctx, db, g, Result{
			GroupName: g.Name, StartedAt: started, CompletedAt: time.Now().UTC(),
			Subdomains: map[string]SubdomainCounts{},
		}, t.Id(), "failure")
	}
	rev, _ := src.Revision(roots[0])

	subCounts := make(map[string]SubdomainCounts, len(g.Subdomains))
	var mu sync.Mutex

	eg, gctx := errgroup.WithContext(ctx)
	for _, sd := range g.Subdomains {
		sd := sd
		eg.Go(func() error {
			counts := runSubdomain(gctx, db, src, roots[0], t, sd)
			mu.Lock()
			subCounts[sd.Name()] = counts
			mu.Unlock()
			return nil
		})
	}
	_ = eg.Wait()

	completed := time.Now().UTC()
	res := Result{
		GroupName:       g.Name,
		CatalogRevision: rev,
		Subdomains:      subCounts,
		StartedAt:       started,
		CompletedAt:     completed,
	}
	outcome := classifyOutcome(subCounts)
	ObserveSeederRun(serviceLabel(), g.Name, outcome, completed.Sub(started).Seconds())
	return persistAndReturn(ctx, db, g, res, t.Id(), outcome)
}

func runSubdomain(ctx context.Context, db *gorm.DB, src CatalogSource, root string, t tenant.Model, sd SubdomainAny) SubdomainCounts {
	var counts SubdomainCounts
	deleted, err := sd.DeleteAllForTenant(db.WithContext(ctx))
	if err != nil {
		counts.Errors = appendError(counts.Errors, fmt.Sprintf("delete: %v", err))
		return counts
	}
	counts.Deleted = deleted

	files, err := src.Walk(root, sd.Path())
	if err != nil {
		counts.Errors = appendError(counts.Errors, fmt.Sprintf("walk %s: %v", sd.Path(), err))
		return counts
	}

	pattern := sd.EntityIDPattern()
	for _, name := range files {
		if err := ctx.Err(); err != nil {
			counts.Errors = appendError(counts.Errors, fmt.Sprintf("%s: %v", name, err))
			return counts
		}
		rows, err := loadOne(ctx, src, root, t, sd, pattern, name)
		if err != nil {
			counts.Failed++
			counts.Errors = appendError(counts.Errors, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		if err := sd.BulkCreate(db.WithContext(ctx), rows); err != nil {
			counts.Failed++
			counts.Errors = appendError(counts.Errors, fmt.Sprintf("%s: bulkcreate: %v", name, err))
			continue
		}
		counts.Created += rowCount(rows)
	}
	return counts
}

func loadOne(ctx context.Context, src CatalogSource, root string, t tenant.Model, sd SubdomainAny, pattern *regexp.Regexp, filename string) (any, error) {
	rc, err := src.Open(root, path.Join(sd.Path(), filename))
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	env, err := ParseEnvelope(b)
	if err != nil {
		return nil, err
	}
	if env.Data.Type != sd.Type() {
		return nil, fmt.Errorf("type mismatch: file has %q, expected %q", env.Data.Type, sd.Type())
	}
	var entityID string
	if pattern != nil {
		id, err := ExtractEntityID(filename, pattern)
		if err != nil {
			return nil, err
		}
		if id != env.Data.ID {
			return nil, fmt.Errorf("id mismatch: filename %q, data.id %q", id, env.Data.ID)
		}
		entityID = id
	} else {
		entityID = env.Data.ID
	}
	// Hand the full file payload to the subdomain rather than just
	// env.Data.Attributes. Most subdomains only need attributes (use
	// seeder.DecodeAttributes helper), but JSON:API files where the
	// per-entity data lives under relationships + included[] (e.g.
	// reactor-drop) need the full envelope to materialize their
	// JSONModel. Type/id checks above ran against the parsed envelope,
	// so passing raw bytes does not lose any validation.
	return sd.LoadAndBuild(t, entityID, b)
}

func rowCount(rows any) int64 {
	v := reflect.ValueOf(rows)
	if v.Kind() != reflect.Slice {
		return 0
	}
	return int64(v.Len())
}

func appendError(in []string, msg string) []string {
	if len(in) >= MaxErrors {
		return in
	}
	return append(in, msg)
}

func classifyOutcome(counts map[string]SubdomainCounts) string {
	if len(counts) == 0 {
		return "failure"
	}
	successCount, failCount := 0, 0
	for _, c := range counts {
		if c.Failed > 0 || len(c.Errors) > 0 {
			failCount++
		} else {
			successCount++
		}
	}
	switch {
	case failCount == 0:
		return "success"
	case successCount == 0:
		return "failure"
	default:
		return "partial"
	}
}

func persistAndReturn(ctx context.Context, db *gorm.DB, g Group, res Result, tenantID uuid.UUID, _ string) (Result, error) {
	summary, err := json.Marshal(res)
	if err != nil {
		return res, fmt.Errorf("marshal summary: %w", err)
	}
	row := SeedState{
		TenantID:        tenantID,
		GroupName:       g.Name,
		CatalogRevision: res.CatalogRevision,
		SeededAt:        res.CompletedAt,
		ResultSummary:   datatypes.JSON(summary),
	}
	if err := UpsertSeedState(db.WithContext(ctx), &row); err != nil {
		return res, fmt.Errorf("upsert seed_state: %w", err)
	}
	return res, nil
}

func serviceLabel() string {
	if v := os.Getenv("ATLAS_SERVICE_NAME"); v != "" {
		return v
	}
	return "unknown"
}
