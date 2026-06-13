package document

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
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
// back to the canonical (uuid.Nil) dataset — mirroring per-id ByIdProvider.
// This is the regression guard for the v84 batch-validation bug: a version
// provisioned after canonical ingestion had zero rows under its own tenant id,
// so GetAll returned empty and preset skill validation rejected every skill.
func TestAllProvider_FallsBackToCanonicalWhenTenantEmpty(t *testing.T) {
	db := newStorageTestDB(t)
	// Canonical rows live under uuid.Nil.
	seedTestDoc(t, db, ctxForTenant(t, uuid.Nil), 1000000, "canonical-a")
	seedTestDoc(t, db, ctxForTenant(t, uuid.Nil), 1000001, "canonical-b")

	// A real tenant with no rows of its own.
	realTenant := uuid.New()
	got, err := newTestStorage(t, db).GetAll(ctxForTenant(t, realTenant))
	require.NoError(t, err)
	require.Len(t, got, 2, "expected fallback to canonical dataset")
}

// When the requesting tenant has its own documents, GetAll must return those
// and must NOT fall back to canonical.
func TestAllProvider_PrefersTenantOwnData(t *testing.T) {
	db := newStorageTestDB(t)
	seedTestDoc(t, db, ctxForTenant(t, uuid.Nil), 1000000, "canonical-a")
	seedTestDoc(t, db, ctxForTenant(t, uuid.Nil), 1000001, "canonical-b")

	realTenant := uuid.New()
	seedTestDoc(t, db, ctxForTenant(t, realTenant), 2000000, "tenant-only")

	got, err := newTestStorage(t, db).GetAll(ctxForTenant(t, realTenant))
	require.NoError(t, err)
	require.Len(t, got, 1, "tenant with its own data should not fall back")
	require.Equal(t, uint32(2000000), got[0].Id)
}

// When neither the tenant nor canonical has data, GetAll returns an empty slice
// without error.
func TestAllProvider_EmptyWhenNeitherHasData(t *testing.T) {
	db := newStorageTestDB(t)
	got, err := newTestStorage(t, db).GetAll(ctxForTenant(t, uuid.New()))
	require.NoError(t, err)
	require.Len(t, got, 0)
}
