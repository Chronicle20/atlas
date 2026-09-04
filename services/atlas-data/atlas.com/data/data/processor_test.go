package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// itemMakeSQLiteDriverName is a sqlite3 driver registered once with a
// uuid_generate_v4 SQL function, matching the Postgres default the
// production document.Entity relies on (`default:uuid_generate_v4()`).
// Without it, sqlite has no such function and every document.Storage.Add
// call fails silently (RegisterFileData discards the error), so nothing is
// ever persisted regardless of whether the ITEM_MAKE branch runs.
const itemMakeSQLiteDriverName = "sqlite3_with_uuid_generate_v4"

var registerItemMakeSQLiteDriverOnce sync.Once

func registerItemMakeSQLiteDriver() {
	registerItemMakeSQLiteDriverOnce.Do(func() {
		sql.Register(itemMakeSQLiteDriverName, &sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				return conn.RegisterFunc("uuid_generate_v4", func() string {
					return uuid.New().String()
				}, false)
			},
		})
	})
}

// itemMakeDocumentEntity mirrors document.Entity's "documents" table shape,
// including the composite unique index document.DbStorage.Add's ON CONFLICT
// clause requires. It is a distinct type (not the package's shared
// testDocumentEntity from status_test.go, which omits that index) migrated
// into its own isolated in-memory database so this test's schema can't be
// shadowed by another test's migration of the same shared-cache "documents"
// table.
type itemMakeDocumentEntity struct {
	Id         uuid.UUID       `gorm:"primaryKey;type:text"`
	TenantId   uuid.UUID       `gorm:"type:text;not null;uniqueIndex:idx_documents_tenant_type_docid"`
	Type       string          `gorm:"not null;uniqueIndex:idx_documents_tenant_type_docid"`
	DocumentId uint32          `gorm:"not null;uniqueIndex:idx_documents_tenant_type_docid"`
	Content    json.RawMessage `gorm:"type:text;not null"`
	UpdatedAt  time.Time       `gorm:"autoUpdateTime"`
}

func (e itemMakeDocumentEntity) TableName() string { return "documents" }

// setupItemMakeDB provisions a real sqlite-backed "documents" table matching
// document.Entity's schema (including the unique index its ON CONFLICT
// upsert depends on) plus the uuid_generate_v4 function that
// document.DbStorage.Add relies on for its Postgres default. Each call gets
// its own isolated in-memory database so it can't collide with another
// test's differently-shaped migration of the same table name.
func setupItemMakeDB(t *testing.T) *gorm.DB {
	t.Helper()
	registerItemMakeSQLiteDriver()

	db, err := gorm.Open(sqlite.Dialector{
		DriverName: itemMakeSQLiteDriverName,
		DSN:        fmt.Sprintf("file:itemmake_%s?mode=memory&cache=shared", uuid.New().String()),
	}, &gorm.Config{
		Logger: logger.New(
			logrus.StandardLogger(),
			logger.Config{
				SlowThreshold: time.Second,
				LogLevel:      logger.Silent,
				Colorful:      false,
			},
		),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&itemMakeDocumentEntity{})
	require.NoError(t, err)

	database.RegisterTenantCallbacks(logrus.StandardLogger(), db)

	return db
}

func TestDataUpdatedEventProvider_KeyIsTenantId(t *testing.T) {
	tenantId := "8b8d2bb0-2d1f-46b0-8c1c-1234567890ab"
	p := dataUpdatedEventProvider(tenantId, WorkerMonster, time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
	msgs, err := p()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	if string(msgs[0].Key) != tenantId {
		t.Fatalf("key = %q, want %q", string(msgs[0].Key), tenantId)
	}
}

func TestDataUpdatedEventProvider_BodyShape(t *testing.T) {
	tenantId := "8b8d2bb0-2d1f-46b0-8c1c-1234567890ab"
	completedAt := time.Date(2026, 5, 8, 12, 30, 0, 0, time.UTC)
	p := dataUpdatedEventProvider(tenantId, WorkerMap, completedAt)
	msgs, _ := p()

	var ev event[dataUpdatedEventBody]
	if err := json.Unmarshal(msgs[0].Value, &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.Type != EventTypeDataUpdated {
		t.Fatalf("Type = %q, want %q", ev.Type, EventTypeDataUpdated)
	}
	if ev.Body.TenantId != tenantId {
		t.Fatalf("TenantId = %q", ev.Body.TenantId)
	}
	if ev.Body.Worker != WorkerMap {
		t.Fatalf("Worker = %q", ev.Body.Worker)
	}
	if ev.Body.CompletedAt != "2026-05-08T12:30:00Z" {
		t.Fatalf("CompletedAt = %q, want RFC3339 UTC", ev.Body.CompletedAt)
	}
}

func TestProducerEnabled_DefaultTrue(t *testing.T) {
	// Snapshot + restore env so other tests don't see our state.
	if v, ok := os.LookupEnv("DATA_EVENTS_PRODUCER_ENABLED"); ok {
		defer os.Setenv("DATA_EVENTS_PRODUCER_ENABLED", v)
	} else {
		defer os.Unsetenv("DATA_EVENTS_PRODUCER_ENABLED")
	}
	os.Unsetenv("DATA_EVENTS_PRODUCER_ENABLED")
	if !producerEnabled() {
		t.Fatal("expected default true when unset")
	}
}

func TestProducerEnabled_ExplicitFalse(t *testing.T) {
	t.Setenv("DATA_EVENTS_PRODUCER_ENABLED", "false")
	if producerEnabled() {
		t.Fatal("expected false when DATA_EVENTS_PRODUCER_ENABLED=false")
	}
}

func TestProducerEnabled_UnparseableTrue(t *testing.T) {
	t.Setenv("DATA_EVENTS_PRODUCER_ENABLED", "not-a-bool")
	if !producerEnabled() {
		t.Fatal("expected default true when unparseable")
	}
}

func TestWorkersIncludesItemMake(t *testing.T) {
	if WorkerItemMake != "ITEM_MAKE" {
		t.Fatalf("WorkerItemMake = %q, want %q", WorkerItemMake, "ITEM_MAKE")
	}
	count := 0
	for _, w := range Workers {
		if w == WorkerItemMake {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("Workers contains WorkerItemMake %d times, want 1", count)
	}
	if len(Workers) != 18 {
		t.Fatalf("len(Workers) = %d, want 18", len(Workers))
	}
}

// itemMakeXMLFixture is the minimal ItemMake.img.xml shape the itemmake
// reader expects: a numeric top-level group directory containing a numeric
// item-id directory with the recipe fields.
const itemMakeXMLFixture = `<imgdir name="ItemMake.img">
  <imgdir name="0">
    <imgdir name="04260000">
      <int name="reqLevel" value="0"/>
      <int name="itemNum" value="1"/>
      <int name="meso" value="500"/>
      <imgdir name="recipe">
        <imgdir name="0">
          <int name="item" value="4000000"/>
          <int name="count" value="1"/>
        </imgdir>
      </imgdir>
    </imgdir>
  </imgdir>
</imgdir>`

// TestStartWorkerDispatchesItemMake proves the WorkerItemMake arm of
// StartWorker is actually reached and performs its work, not merely that
// StartWorker returns nil (RegisterFileData discards rf's error, so a
// nil-error assertion alone would pass identically if the branch were
// deleted or no-op'd). It writes a real ItemMake.img.xml fixture, runs the
// worker against a real sqlite-backed document store, and asserts the
// itemmake row that only the RegisterItemMake call could have produced.
func TestStartWorkerDispatchesItemMake(t *testing.T) {
	t.Setenv("DATA_EVENTS_PRODUCER_ENABLED", "false")
	tenantId := uuid.New()
	tn, err := tenant.Create(tenantId, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	ctx := tenant.WithContext(context.Background(), tn)

	db := setupItemMakeDB(t)

	tmp := t.TempDir()
	etcDir := filepath.Join(tmp, "Etc.wz")
	if err = os.MkdirAll(etcDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err = os.WriteFile(filepath.Join(etcDir, "ItemMake.img.xml"), []byte(itemMakeXMLFixture), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	p := &ProcessorImpl{l: l, ctx: ctx, db: db}
	if err = p.StartWorker(WorkerItemMake, tmp); err != nil {
		t.Fatalf("StartWorker(WorkerItemMake, ...) returned error, want nil: %v", err)
	}

	var row itemMakeDocumentEntity
	if err = db.WithContext(ctx).Where("tenant_id = ? AND type = ? AND document_id = ?", tenantId, "ITEM_MAKE", 4260000).First(&row).Error; err != nil {
		t.Fatalf("expected an ITEM_MAKE document for item 4260000 to have been persisted by the ITEM_MAKE worker branch, got: %v", err)
	}
}
