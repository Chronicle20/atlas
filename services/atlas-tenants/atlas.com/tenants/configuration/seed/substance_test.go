package seed_test

import (
	"atlas-tenants/configuration"
	"atlas-tenants/configuration/seed"
	"atlas-tenants/test"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// repoRootDeploySeedEnvVar is the env var name seedSource points
// seeder.NewFilesystemCatalogSourceWithShared at. Using a dedicated,
// test-only var name (rather than SEED_CATALOG_ROOT, which InitResource
// uses) keeps these tests independent of the process environment.
const repoRootDeploySeedEnvVar = "SEED_CATALOG_ROOT_SUBSTANCE_TEST"

// repoRootDeploySeed resolves the absolute path to <repo-root>/deploy/seed
// from this test file's location, so these tests exercise the real shared
// catalog (deploy/seed/shared/all/...) rather than a fixture copy.
func repoRootDeploySeed(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get caller info")
	}
	// substance_test.go is in configuration/seed/; deploy/seed lives at the
	// repo root, 6 levels up: seed -> configuration -> tenants -> atlas.com
	// -> atlas-tenants -> services -> repo root.
	root := filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "..", "..", "deploy", "seed")
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("resolve deploy/seed path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(abs, "shared", "all")); err != nil {
		t.Fatalf("deploy/seed/shared/all not found at %s: %v", abs, err)
	}
	return abs
}

func newSubstanceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := test.SetupTestDB(t)
	t.Cleanup(func() { test.CleanupTestDB(db) })
	if err := db.AutoMigrate(&seeder.SeedState{}); err != nil {
		t.Fatalf("migrate seed_state: %v", err)
	}
	if err := outbox.Migration(db); err != nil {
		t.Fatalf("migrate outbox: %v", err)
	}
	return db
}

func newSubstanceTestLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func newSubstanceTenant(t *testing.T) tenant.Model {
	t.Helper()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	return te
}

// instanceRoutesGroup returns the "instance-routes" seeder.Group produced by
// seed.Groups — the same Group InitResource wires to the HTTP routes — so
// these tests exercise the real AfterSeed hook, not a hand-rolled stand-in.
func instanceRoutesGroup(t *testing.T, l logrus.FieldLogger) seeder.Group {
	t.Helper()
	for _, g := range seed.Groups(l) {
		if g.Name == "instance-routes" {
			return g
		}
	}
	t.Fatal("instance-routes group not found in seed.Groups")
	return seeder.Group{}
}

// seedSynchronously points a filesystem CatalogSource at the real
// deploy/seed/shared/all tree, runs seeder.Seed synchronously (bypassing the
// background-goroutine path the HTTP handler uses, per
// atlas-reward-pools/seed/groups_test.go's TestSeed_IncubatorKindAndWeightRoundTrip
// pattern), and then invokes the group's AfterSeed hook exactly the way
// libs/atlas-seeder's postSeed does after a successful Seed.
func seedSynchronously(t *testing.T, db *gorm.DB, l logrus.FieldLogger, g seeder.Group, te tenant.Model) seeder.Result {
	t.Helper()
	t.Setenv(repoRootDeploySeedEnvVar, repoRootDeploySeed(t))
	src := seeder.NewFilesystemCatalogSourceWithShared(repoRootDeploySeedEnvVar, "unused-fallback", "shared/all")

	ctx := tenant.WithContext(context.Background(), te)
	res, err := seeder.Seed(ctx, db, src, g)
	if err != nil {
		t.Fatalf("seeder.Seed: %v", err)
	}
	if g.AfterSeed != nil {
		if err := g.AfterSeed(ctx, db, res); err != nil {
			t.Fatalf("AfterSeed: %v", err)
		}
	}
	return res
}

// TestSeed_InstanceRoutesRoundTrip seeds the real
// deploy/seed/shared/all/instance-routes catalog synchronously and verifies:
//  1. the resulting entry count matches the on-disk file count;
//  2. an entry whose filename differs from its id
//     (flight-temple-of-time-leafre.json -> id
//     temple-of-time-return-flight) is stored keyed by data.id, not by any
//     filename-derived string.
func TestSeed_InstanceRoutesRoundTrip(t *testing.T) {
	db := newSubstanceTestDB(t)
	l := newSubstanceTestLogger()
	g := instanceRoutesGroup(t, l)
	te := newSubstanceTenant(t)

	catalogDir := filepath.Join(repoRootDeploySeed(t), "shared", "all", "instance-routes")
	entries, err := os.ReadDir(catalogDir)
	if err != nil {
		t.Fatalf("read catalog dir: %v", err)
	}
	fileCount := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			fileCount++
		}
	}
	if fileCount == 0 {
		t.Fatal("no .json files found in deploy/seed/shared/all/instance-routes; fixture assumption broke")
	}

	seedSynchronously(t, db, l, g, te)

	all, err := configuration.GetAllInstanceRoutesProvider(te.Id())(db)()
	if err != nil {
		t.Fatalf("GetAllInstanceRoutesProvider: %v", err)
	}
	if len(all) != fileCount {
		t.Errorf("seeded entry count = %d, want %d (on-disk .json file count)", len(all), fileCount)
	}

	// flight-temple-of-time-leafre.json holds data.id
	// "temple-of-time-return-flight" — the entry must be addressable by
	// that id, not by the filename stem.
	byRealId, err := configuration.GetInstanceRouteByIdProvider(te.Id(), "temple-of-time-return-flight")(db)()
	if err != nil {
		t.Errorf("GetInstanceRouteByIdProvider(temple-of-time-return-flight): %v", err)
	}
	if id, _ := byRealId["id"].(string); id != "temple-of-time-return-flight" {
		t.Errorf("entry id = %q, want %q", id, "temple-of-time-return-flight")
	}

	if _, err := configuration.GetInstanceRouteByIdProvider(te.Id(), "flight-temple-of-time-leafre")(db)(); err == nil {
		t.Error("GetInstanceRouteByIdProvider(flight-temple-of-time-leafre) unexpectedly succeeded — entry must not be keyed by the filename stem")
	}
}

// TestSeed_ExactlyOneOutboxRowPerRun verifies the plan's central correctness
// claim for AfterSeed: seeding a resource with many files (instance-routes,
// currently a dozen+) enqueues exactly ONE outbox row, not one per file. The
// atlas-transports consumer for this event does a full clear-and-reload, so
// N events would trigger N reloads and risk a reload landing mid-seed. The
// single row must also carry the seeding tenant's headers, since AfterSeed
// runs against libs/atlas-seeder's tenant-bearing background context.
func TestSeed_ExactlyOneOutboxRowPerRun(t *testing.T) {
	db := newSubstanceTestDB(t)
	l := newSubstanceTestLogger()
	g := instanceRoutesGroup(t, l)
	te := newSubstanceTenant(t)

	seedSynchronously(t, db, l, g, te)

	var rows []outbox.Entity
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("query outbox_entries: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("outbox_entries row count = %d, want 1 (one event per seed run, not one per file)", len(rows))
	}

	// libs/atlas-outbox base64-encodes header values at rest (headers.go's
	// encodeHeaders/decodeHeaders) since Kafka header values are raw bytes;
	// decode before comparing.
	var encoded map[string]string
	if err := json.Unmarshal(rows[0].Headers, &encoded); err != nil {
		t.Fatalf("unmarshal outbox headers: %v", err)
	}
	headers := make(map[string]string, len(encoded))
	for k, v := range encoded {
		b, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			t.Fatalf("base64-decode header %q: %v", k, err)
		}
		headers[k] = string(b)
	}
	if got := headers[tenant.ID]; got != te.Id().String() {
		t.Errorf("outbox row tenant.ID header = %q, want %q", got, te.Id().String())
	}
	if got := headers[tenant.Region]; got != te.Region() {
		t.Errorf("outbox row tenant.Region header = %q, want %q", got, te.Region())
	}
}

// TestSeed_TenantIsolation verifies seeding tenant A's instance-routes
// leaves tenant B's count at 0 — Subdomain.BulkCreate/Count/
// DeleteAllForTenant must all scope strictly to the seeding tenant.
func TestSeed_TenantIsolation(t *testing.T) {
	db := newSubstanceTestDB(t)
	l := newSubstanceTestLogger()
	g := instanceRoutesGroup(t, l)
	tenantA := newSubstanceTenant(t)
	tenantB := newSubstanceTenant(t)

	seedSynchronously(t, db, l, g, tenantA)

	countA, _, err := configuration.CountConfigurationEntries(db, tenantA.Id(), "instance-routes")
	if err != nil {
		t.Fatalf("CountConfigurationEntries(tenantA): %v", err)
	}
	if countA == 0 {
		t.Fatal("tenant A count = 0, want > 0 after seeding tenant A")
	}

	countB, _, err := configuration.CountConfigurationEntries(db, tenantB.Id(), "instance-routes")
	if err != nil {
		t.Fatalf("CountConfigurationEntries(tenantB): %v", err)
	}
	if countB != 0 {
		t.Errorf("tenant B count = %d, want 0 — seeding tenant A must not leak into tenant B", countB)
	}
}

// sumSubdomainCounts totals Created/Deleted across every subdomain in a
// seeder.Result. Groups.go's newGroup wires exactly one subdomain per group,
// but summing rather than indexing keeps these tests correct even if that
// changes.
func sumSubdomainCounts(subs map[string]seeder.SubdomainCounts) (created, deleted int64) {
	for _, c := range subs {
		created += c.Created
		deleted += c.Deleted
	}
	return created, deleted
}

// TestSeed_AfterSeedGuardSkipsEmitOnDeleteWithoutCreate is the regression
// test for the final-review finding: libs/atlas-seeder's runSubdomain
// deletes the tenant's entire resource row set BEFORE walking the catalog,
// and Walk returns (nil, nil) for a missing directory, so a seed run
// against an absent/unmounted catalog mount scores Created=0, Deleted=N and
// classifyOutcome still reports "success". Without a guard, afterSeed would
// emit a configuration-status event telling atlas-transports' ClearTenant +
// full-reload consumer to reload a now-empty resource, wiping the live
// Redis route registry. This test seeds real data first, then re-seeds
// against an empty stand-in catalog root and asserts no second outbox row
// is enqueued.
func TestSeed_AfterSeedGuardSkipsEmitOnDeleteWithoutCreate(t *testing.T) {
	db := newSubstanceTestDB(t)
	l := newSubstanceTestLogger()
	g := instanceRoutesGroup(t, l)
	te := newSubstanceTenant(t)

	// First seed against the real catalog creates rows and enqueues the
	// one outbox row a normal successful seed must still produce.
	seedSynchronously(t, db, l, g, te)

	var afterFirst []outbox.Entity
	if err := db.Find(&afterFirst).Error; err != nil {
		t.Fatalf("query outbox after first (real-catalog) seed: %v", err)
	}
	if len(afterFirst) != 1 {
		t.Fatalf("outbox rows after first seed = %d, want 1", len(afterFirst))
	}

	// Second seed points at an empty temp directory standing in for a
	// missing/unmounted catalog mount. DeleteAllForTenant still runs
	// unconditionally and removes every row the first seed created;
	// Walk returns (nil, nil) for the missing subdirectory so nothing is
	// created. Run seeder.Seed directly (not via seedSynchronously) since
	// AfterSeed returning an error here is exactly what's under test.
	emptyRoot := t.TempDir()
	const emptyRootEnvVar = "SEED_CATALOG_ROOT_SUBSTANCE_TEST_EMPTY"
	t.Setenv(emptyRootEnvVar, emptyRoot)
	src := seeder.NewFilesystemCatalogSourceWithShared(emptyRootEnvVar, "unused-fallback", "shared/all")

	ctx := tenant.WithContext(context.Background(), te)
	res, err := seeder.Seed(ctx, db, src, g)
	if err != nil {
		t.Fatalf("seeder.Seed (empty catalog): %v", err)
	}
	created, deleted := sumSubdomainCounts(res.Subdomains)
	if created != 0 || deleted == 0 {
		t.Fatalf("second seed created=%d deleted=%d, want created=0 deleted>0 (fixture assumption broke)", created, deleted)
	}

	if g.AfterSeed == nil {
		t.Fatal("group AfterSeed is nil")
	}
	if err := g.AfterSeed(ctx, db, res); err == nil {
		t.Error("AfterSeed(created=0, deleted>0) returned nil, want a non-nil error (guard must refuse the emit)")
	}

	var afterSecond []outbox.Entity
	if err := db.Find(&afterSecond).Error; err != nil {
		t.Fatalf("query outbox after second (empty-catalog) seed: %v", err)
	}
	if len(afterSecond) != 1 {
		t.Errorf("outbox rows after empty-catalog seed = %d, want still 1 — the guard must not enqueue a second row", len(afterSecond))
	}
}

// TestSeed_AfterSeedGuardAllowsEmitOnLegitimatelyEmptyFirstSeed verifies the
// guard's boundary condition: Created==0 && Deleted==0 (an unseeded
// tenant's first run against an empty catalog) is NOT the dangerous case —
// nothing existed to lose — and must still emit exactly like any other
// successful seed.
func TestSeed_AfterSeedGuardAllowsEmitOnLegitimatelyEmptyFirstSeed(t *testing.T) {
	db := newSubstanceTestDB(t)
	l := newSubstanceTestLogger()
	g := instanceRoutesGroup(t, l)
	te := newSubstanceTenant(t)

	emptyRoot := t.TempDir()
	const emptyRootEnvVar = "SEED_CATALOG_ROOT_SUBSTANCE_TEST_EMPTY_FIRST"
	t.Setenv(emptyRootEnvVar, emptyRoot)
	src := seeder.NewFilesystemCatalogSourceWithShared(emptyRootEnvVar, "unused-fallback", "shared/all")

	ctx := tenant.WithContext(context.Background(), te)
	res, err := seeder.Seed(ctx, db, src, g)
	if err != nil {
		t.Fatalf("seeder.Seed (empty catalog, unseeded tenant): %v", err)
	}
	created, deleted := sumSubdomainCounts(res.Subdomains)
	if created != 0 || deleted != 0 {
		t.Fatalf("first seed created=%d deleted=%d, want created=0 deleted=0 (fixture assumption broke)", created, deleted)
	}

	if err := g.AfterSeed(ctx, db, res); err != nil {
		t.Errorf("AfterSeed(created=0, deleted=0) returned error %v, want nil — a legitimately-empty first seed must not be treated as an error", err)
	}

	var rows []outbox.Entity
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("query outbox: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("outbox rows after legitimately-empty first seed = %d, want 1 — the guard must still emit", len(rows))
	}
}
