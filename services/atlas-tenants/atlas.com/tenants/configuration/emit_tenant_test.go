package configuration_test

import (
	"atlas-tenants/configuration"
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	tenants "atlas-tenants/tenant"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	atlastenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// decodeOutboxHeaders undoes the base64 encoding libs/atlas-outbox applies
// to every header value before storing it in the jsonb `headers` column
// (see headers.go: version headers are raw big-endian uint16 bytes, which
// contain NUL/invalid-UTF8 bytes that jsonb and encoding/json can't carry
// verbatim, so ALL header values — not just the version ones — are
// base64-encoded uniformly).
func decodeOutboxHeaders(t *testing.T, raw []byte) map[string]string {
	t.Helper()
	var enc map[string]string
	if err := json.Unmarshal(raw, &enc); err != nil {
		t.Fatalf("decode headers: %v", err)
	}
	dec := make(map[string]string, len(enc))
	for k, v := range enc {
		b, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			t.Fatalf("base64-decode header %s: %v", k, err)
		}
		dec[k] = string(b)
	}
	return dec
}

// newEmitTestDB adds the outbox table to the shared test database
// (test.SetupTestDB already migrates tenant.Entity + configuration.Entity)
// so an …AndEmit call can be observed at the outbox row it writes. This is
// the same pattern rankings_handler_test.go already uses.
func newEmitTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := configTestDB(t) // from administrator_test.go
	if err := outbox.Migration(db); err != nil {
		t.Fatalf("migrate outbox: %v", err)
	}
	return db
}

// seedTenantRow inserts a real tenant row for tenantCtx's
// tenant.Processor.GetById to resolve. The local tenant package's Create
// takes a *message.Buffer (only callable inside a transaction) and
// CreateAndEmit itself writes an outbox row — which would pollute the
// "exactly one outbox row" assertions the …AndEmit tests below make against
// the configuration event. Inserting the entity directly avoids that
// side-effect while still exercising tenantCtx's real GetById -> Create ->
// WithContext path.
func seedTenantRow(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	e := tenants.Entity{
		ID:           uuid.New(),
		Name:         "emit-test",
		Region:       "GMS",
		MajorVersion: 83,
		MinorVersion: 1,
	}
	if err := db.Create(&e).Error; err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	return e.ID
}

// The regression this task exists for: a configuration-status event must
// carry all four tenant headers. Asserting at the outbox row covers the
// enqueue-time header snapshot that actually failed in production.
func TestCreateRouteAndEmit_OutboxRowCarriesTenantHeaders(t *testing.T) {
	db := newEmitTestDB(t)
	tid := seedTenantRow(t, db)

	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	p := configuration.NewProcessor(l, context.Background(), db)

	if _, err := p.CreateRouteAndEmit(tid, map[string]interface{}{
		"id":         "boat-ellinia-orbis",
		"type":       "routes",
		"attributes": map[string]interface{}{"name": "boat-ellinia-orbis"},
	}); err != nil {
		t.Fatalf("CreateRouteAndEmit: %v", err)
	}

	var rows []outbox.Entity
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("read outbox: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("outbox rows = %d, want 1", len(rows))
	}

	headers := decodeOutboxHeaders(t, rows[0].Headers)
	if headers[atlastenant.ID] != tid.String() {
		t.Errorf("%s = %q, want %q", atlastenant.ID, headers[atlastenant.ID], tid.String())
	}
	if headers[atlastenant.Region] != "GMS" {
		t.Errorf("%s = %q, want %q", atlastenant.Region, headers[atlastenant.Region], "GMS")
	}
	if _, ok := headers[atlastenant.MajorVersion]; !ok {
		t.Errorf("%s header missing", atlastenant.MajorVersion)
	}
	if _, ok := headers[atlastenant.MinorVersion]; !ok {
		t.Errorf("%s header missing", atlastenant.MinorVersion)
	}
}

// An unknown tenant must abort the write rather than emit tenant-free:
// the operation is meaningless for a tenant that does not exist, and a
// tenant-free emit is exactly the defect being closed.
func TestCreateRouteAndEmit_UnknownTenantFails(t *testing.T) {
	db := newEmitTestDB(t)
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	p := configuration.NewProcessor(l, context.Background(), db)

	if _, err := p.CreateRouteAndEmit(uuid.New(), map[string]interface{}{
		"id":         "ghost",
		"type":       "routes",
		"attributes": map[string]interface{}{"name": "ghost"},
	}); err == nil {
		t.Fatal("CreateRouteAndEmit(unknown tenant) returned nil error, want failure")
	}

	var count int64
	if err := db.Model(&outbox.Entity{}).Count(&count).Error; err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if count != 0 {
		t.Fatalf("outbox rows = %d, want 0 (nothing may be emitted for an unknown tenant)", count)
	}
}

// Rankings emits through the same tenant-free context and was verified
// during design to share the defect (processor.go CreateRankingsAndEmit).
func TestCreateRankingsAndEmit_OutboxRowCarriesTenantHeaders(t *testing.T) {
	db := newEmitTestDB(t)
	tid := seedTenantRow(t, db)

	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	p := configuration.NewProcessor(l, context.Background(), db)

	if _, err := p.CreateRankingsAndEmit(tid, map[string]interface{}{
		"id":         "rankings",
		"type":       "rankings",
		"attributes": map[string]interface{}{},
	}); err != nil {
		t.Fatalf("CreateRankingsAndEmit: %v", err)
	}

	var rows []outbox.Entity
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("read outbox: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("outbox rows = %d, want 1", len(rows))
	}
	headers := decodeOutboxHeaders(t, rows[0].Headers)
	if headers[atlastenant.ID] != tid.String() {
		t.Errorf("%s = %q, want %q", atlastenant.ID, headers[atlastenant.ID], tid.String())
	}
}
