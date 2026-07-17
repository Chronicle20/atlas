package document

import (
	"atlas-data/canonical"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testEntity mirrors document.Entity without the PostgreSQL-specific
// uuid_generate_v4() column default so it migrates cleanly under sqlite.
type testEntity struct {
	Id         uuid.UUID       `gorm:"primaryKey;type:text"`
	TenantId   uuid.UUID       `gorm:"type:text;not null;uniqueIndex:idx_documents_tenant_type_docid"`
	Type       string          `gorm:"not null;uniqueIndex:idx_documents_tenant_type_docid"`
	DocumentId uint32          `gorm:"not null;uniqueIndex:idx_documents_tenant_type_docid"`
	Content    json.RawMessage `gorm:"type:text;not null"`
	UpdatedAt  time.Time       `gorm:"autoUpdateTime"`
}

func (e testEntity) TableName() string { return "documents" }

// testDoc is a minimal jsonapi-marshalable document model used to exercise Storage.
type testDoc struct {
	Id   uint32 `json:"-"`
	Name string `json:"name"`
}

func (m testDoc) GetName() string { return "testdocs" }
func (m testDoc) GetID() string   { return fmt.Sprint(m.Id) }
func (m *testDoc) SetID(id string) error {
	var x uint32
	if _, err := fmt.Sscan(id, &x); err != nil {
		return err
	}
	m.Id = x
	return nil
}

const testDocType = "TESTDOC"

func newStorageTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Per-test named in-memory DB (shared cache keeps the schema visible across
	// the gorm connection pool) so count-based assertions don't bleed between tests.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.New(logrus.StandardLogger(), logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Silent,
		}),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&testEntity{}))
	database.RegisterTenantCallbacks(logrus.StandardLogger(), db)
	return db
}

func ctxForTenant(t *testing.T, id uuid.UUID) context.Context {
	t.Helper()
	tn, err := tenant.Create(id, "GMS", 84, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tn)
}

// ctxForVersionedTenant returns a context carrying a tenant with the given id,
// region, major, and minor version. Used by version-aware fallback tests.
func ctxForVersionedTenant(t *testing.T, id uuid.UUID, region string, major, minor uint16) context.Context {
	t.Helper()
	tn, err := tenant.Create(id, region, major, minor)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tn)
}

func newTestStorage(t *testing.T, db *gorm.DB) *Storage[string, testDoc] {
	t.Helper()
	return NewStorage[string, testDoc](logrus.StandardLogger(), db, NewRegistry[string, testDoc](), testDocType)
}

// seedTestDoc inserts a single document scoped to the tenant carried by ctx.
func seedTestDoc(t *testing.T, db *gorm.DB, ctx context.Context, id uint32, name string) {
	t.Helper()
	_, err := newTestStorage(t, db).Add(ctx)(testDoc{Id: id, Name: name})()
	require.NoError(t, err)
}

// When the requesting tenant has no documents of this type, GetAll must fall
// back to the version-scoped canonical dataset — mirroring per-id ByIdProvider.
// This is the regression guard for the v84 batch-validation bug: a version
// provisioned after canonical ingestion had zero rows under its own tenant id,
// so GetAll returned empty and preset skill validation rejected every skill.
// ctxForTenant uses "GMS" 84 1, so the canonical fallback id is canonical.TenantId("GMS",84,1).
func TestAllProvider_FallsBackToCanonicalWhenTenantEmpty(t *testing.T) {
	db := newStorageTestDB(t)
	// Canonical rows live under the version-scoped canonical id (GMS 84.1 — matches ctxForTenant).
	canonId := canonical.TenantId("GMS", 84, 1)
	canonCtx := ctxForVersionedTenant(t, canonId, "GMS", 84, 1)
	seedTestDoc(t, db, canonCtx, 1000000, "canonical-a")
	seedTestDoc(t, db, canonCtx, 1000001, "canonical-b")

	// A real tenant with no rows of its own.
	realTenant := uuid.New()
	got, err := newTestStorage(t, db).DrainAllProvider(ctxForTenant(t, realTenant))()
	require.NoError(t, err)
	require.Len(t, got, 2, "expected fallback to canonical dataset")
}

// When the requesting tenant has its own documents, GetAll must return those
// and must NOT fall back to canonical.
func TestAllProvider_PrefersTenantOwnData(t *testing.T) {
	db := newStorageTestDB(t)
	canonId := canonical.TenantId("GMS", 84, 1)
	canonCtx := ctxForVersionedTenant(t, canonId, "GMS", 84, 1)
	seedTestDoc(t, db, canonCtx, 1000000, "canonical-a")
	seedTestDoc(t, db, canonCtx, 1000001, "canonical-b")

	realTenant := uuid.New()
	seedTestDoc(t, db, ctxForTenant(t, realTenant), 2000000, "tenant-only")

	got, err := newTestStorage(t, db).DrainAllProvider(ctxForTenant(t, realTenant))()
	require.NoError(t, err)
	require.Len(t, got, 1, "tenant with its own data should not fall back")
	require.Equal(t, uint32(2000000), got[0].Id)
}

// When neither the tenant nor canonical has data, GetAll returns an empty slice
// without error.
func TestAllProvider_EmptyWhenNeitherHasData(t *testing.T) {
	db := newStorageTestDB(t)
	got, err := newTestStorage(t, db).DrainAllProvider(ctxForTenant(t, uuid.New()))()
	require.NoError(t, err)
	require.Len(t, got, 0)
}

// Version-aware canonical fallback: v83 and v84 tenants with no rows of their
// own should each fall back to their respective version-scoped canonical dataset
// (seeded under canonical.TenantId), not bleed into each other.
func TestVersionScopedCanonicalFallback_GetAll(t *testing.T) {
	db := newStorageTestDB(t)

	// Seed v83 canonical rows under canonical.TenantId("GMS", 83, 1).
	v83CanId := canonical.TenantId("GMS", 83, 1)
	v83CanCtx := ctxForVersionedTenant(t, v83CanId, "GMS", 83, 1)
	seedTestDoc(t, db, v83CanCtx, 8300000, "v83-canonical-a")
	seedTestDoc(t, db, v83CanCtx, 8300001, "v83-canonical-b")

	// Seed v84 canonical rows under canonical.TenantId("GMS", 84, 1) — different content.
	v84CanId := canonical.TenantId("GMS", 84, 1)
	v84CanCtx := ctxForVersionedTenant(t, v84CanId, "GMS", 84, 1)
	seedTestDoc(t, db, v84CanCtx, 8400000, "v84-canonical-a")

	// A v83 tenant with no per-tenant rows should fall back to v83 canonical data.
	v83TenantCtx := ctxForVersionedTenant(t, uuid.New(), "GMS", 83, 1)
	got83, err := newTestStorage(t, db).DrainAllProvider(v83TenantCtx)()
	require.NoError(t, err)
	require.Len(t, got83, 2, "v83 tenant should fall back to v83 canonical dataset")
	ids83 := make(map[uint32]bool)
	for _, d := range got83 {
		ids83[d.Id] = true
	}
	require.True(t, ids83[8300000], "v83 fallback should contain v83 doc 8300000")
	require.True(t, ids83[8300001], "v83 fallback should contain v83 doc 8300001")

	// A v84 tenant with no per-tenant rows should fall back to v84 canonical data (no cross-version bleed).
	v84TenantCtx := ctxForVersionedTenant(t, uuid.New(), "GMS", 84, 1)
	got84, err := newTestStorage(t, db).DrainAllProvider(v84TenantCtx)()
	require.NoError(t, err)
	require.Len(t, got84, 1, "v84 tenant should fall back to v84 canonical dataset only")
	require.Equal(t, uint32(8400000), got84[0].Id, "v84 fallback should not contain v83 docs")
}

// Version-aware canonical fallback: GetById should resolve the correct
// version-scoped canonical row when the tenant has no per-tenant rows.
func TestVersionScopedCanonicalFallback_GetById(t *testing.T) {
	db := newStorageTestDB(t)

	// Seed v83 canonical row under canonical.TenantId("GMS", 83, 1).
	v83CanId := canonical.TenantId("GMS", 83, 1)
	v83CanCtx := ctxForVersionedTenant(t, v83CanId, "GMS", 83, 1)
	seedTestDoc(t, db, v83CanCtx, 9300000, "v83-by-id")

	// Seed v84 canonical row with a different name under the same document id.
	v84CanId := canonical.TenantId("GMS", 84, 1)
	v84CanCtx := ctxForVersionedTenant(t, v84CanId, "GMS", 84, 1)
	seedTestDoc(t, db, v84CanCtx, 9300000, "v84-by-id")

	// v83 tenant gets v83 content.
	v83TenantCtx := ctxForVersionedTenant(t, uuid.New(), "GMS", 83, 1)
	got83, err := newTestStorage(t, db).GetById(v83TenantCtx)("9300000")
	require.NoError(t, err)
	require.Equal(t, "v83-by-id", got83.Name, "v83 GetById should return v83 canonical content")

	// v84 tenant gets v84 content (no cross-version bleed).
	v84TenantCtx := ctxForVersionedTenant(t, uuid.New(), "GMS", 84, 1)
	got84, err := newTestStorage(t, db).GetById(v84TenantCtx)("9300000")
	require.NoError(t, err)
	require.Equal(t, "v84-by-id", got84.Name, "v84 GetById should return v84 canonical content, not v83")
}

// A tenant that has its own per-tenant rows still reads its own data (fallback not taken).
func TestVersionScopedCanonicalFallback_OwnDataPreferred(t *testing.T) {
	db := newStorageTestDB(t)

	// Seed v83 canonical.
	v83CanId := canonical.TenantId("GMS", 83, 1)
	v83CanCtx := ctxForVersionedTenant(t, v83CanId, "GMS", 83, 1)
	seedTestDoc(t, db, v83CanCtx, 5000000, "canonical-name")

	// The real tenant seeds its own row with a different name.
	realId := uuid.New()
	realCtx := ctxForVersionedTenant(t, realId, "GMS", 83, 1)
	seedTestDoc(t, db, realCtx, 5000000, "tenant-name")

	gotAll, err := newTestStorage(t, db).DrainAllProvider(realCtx)()
	require.NoError(t, err)
	require.Len(t, gotAll, 1)
	require.Equal(t, "tenant-name", gotAll[0].Name, "tenant's own data should be preferred over canonical fallback")

	gotById, err := newTestStorage(t, db).GetById(realCtx)("5000000")
	require.NoError(t, err)
	require.Equal(t, "tenant-name", gotById.Name, "GetById: tenant's own data should be preferred over canonical fallback")
}

// GetById and GetAll agree for the same tenant (version-scoped canonical fallback).
func TestVersionScopedCanonicalFallback_GetByIdAndGetAllAgree(t *testing.T) {
	db := newStorageTestDB(t)

	// Seed v83 canonical with two docs.
	v83CanId := canonical.TenantId("GMS", 83, 1)
	v83CanCtx := ctxForVersionedTenant(t, v83CanId, "GMS", 83, 1)
	seedTestDoc(t, db, v83CanCtx, 7000000, "agree-a")
	seedTestDoc(t, db, v83CanCtx, 7000001, "agree-b")

	// v83 tenant with no rows of its own.
	tenantCtx := ctxForVersionedTenant(t, uuid.New(), "GMS", 83, 1)
	sto := newTestStorage(t, db)

	all, err := sto.DrainAllProvider(tenantCtx)()
	require.NoError(t, err)
	require.Len(t, all, 2)

	byId0, err := sto.GetById(tenantCtx)("7000000")
	require.NoError(t, err)
	require.Equal(t, uint32(7000000), byId0.Id)

	byId1, err := sto.GetById(tenantCtx)("7000001")
	require.NoError(t, err)
	require.Equal(t, uint32(7000001), byId1.Id)
}

// AllPagedProvider must page a tenant's own documents in document_id order,
// with Total reflecting the full unpaged count.
func TestAllPagedProviderPagesTenantDocs(t *testing.T) {
	db := newStorageTestDB(t)
	tenantCtx := ctxForVersionedTenant(t, uuid.New(), "GMS", 83, 1)
	for i := uint32(1); i <= 7; i++ {
		seedTestDoc(t, db, tenantCtx, i, fmt.Sprintf("doc-%d", i))
	}

	p, err := newTestStorage(t, db).AllPagedProvider(tenantCtx)(model.Page{Number: 2, Size: 3})()
	require.NoError(t, err)
	require.Equal(t, 7, p.Total)
	require.Equal(t, model.Page{Number: 2, Size: 3}, p.Page)
	require.Len(t, p.Items, 3)
	require.Equal(t, []uint32{4, 5, 6}, []uint32{p.Items[0].Id, p.Items[1].Id, p.Items[2].Id})
}

// When the requesting tenant has no documents of this type, AllPagedProvider
// must fall back to the version-scoped canonical dataset — mirroring
// AllProvider's regression guard (batch-GetAll-skips-fallback, PR #759) so
// the paged variant does not reintroduce that asymmetry.
func TestAllPagedProviderCanonicalFallbackOnEmpty(t *testing.T) {
	db := newStorageTestDB(t)
	canonId := canonical.TenantId("GMS", 83, 1)
	canonCtx := ctxForVersionedTenant(t, canonId, "GMS", 83, 1)
	for i := uint32(1); i <= 4; i++ {
		seedTestDoc(t, db, canonCtx, i, fmt.Sprintf("canonical-%d", i))
	}

	tenantCtx := ctxForVersionedTenant(t, uuid.New(), "GMS", 83, 1)
	p, err := newTestStorage(t, db).AllPagedProvider(tenantCtx)(model.Page{Number: 1, Size: 10})()
	require.NoError(t, err)
	require.Equal(t, 4, p.Total, "expected fallback to canonical dataset")
	require.Len(t, p.Items, 4)
}

// A tenant with its own documents must not fall back to canonical, even
// though canonical also has rows of this type.
func TestAllPagedProviderNoFallbackWhenTenantHasDocs(t *testing.T) {
	db := newStorageTestDB(t)
	canonId := canonical.TenantId("GMS", 83, 1)
	canonCtx := ctxForVersionedTenant(t, canonId, "GMS", 83, 1)
	for i := uint32(1); i <= 4; i++ {
		seedTestDoc(t, db, canonCtx, i, fmt.Sprintf("canonical-%d", i))
	}

	tenantCtx := ctxForVersionedTenant(t, uuid.New(), "GMS", 83, 1)
	seedTestDoc(t, db, tenantCtx, 100, "tenant-a")
	seedTestDoc(t, db, tenantCtx, 101, "tenant-b")

	p, err := newTestStorage(t, db).AllPagedProvider(tenantCtx)(model.Page{Number: 1, Size: 10})()
	require.NoError(t, err)
	require.Equal(t, 2, p.Total, "tenant with its own data must not fall back to canonical")
	require.Len(t, p.Items, 2)
}

// DrainAllProvider must page internally until every document is collected.
func TestDrainAllProviderReturnsAll(t *testing.T) {
	db := newStorageTestDB(t)
	tenantCtx := ctxForVersionedTenant(t, uuid.New(), "GMS", 83, 1)
	for i := uint32(1); i <= 12; i++ {
		seedTestDoc(t, db, tenantCtx, i, fmt.Sprintf("doc-%d", i))
	}

	all, err := newTestStorage(t, db).DrainAllProvider(tenantCtx)()
	require.NoError(t, err)
	require.Len(t, all, 12)
}
