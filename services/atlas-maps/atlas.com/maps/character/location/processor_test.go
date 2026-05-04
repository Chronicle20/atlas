package location

import (
	"context"
	"errors"
	"testing"

	"atlas-maps/data/map/info"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// stubInfoProcessor lets us inject map data without atlas-data round-trips.
type stubInfoProcessor struct {
	out info.Model
	err error
}

func (s *stubInfoProcessor) GetById(_ _map.Id) (info.Model, error) {
	return s.out, s.err
}

func newCtxTenant(t *testing.T) context.Context {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tn)
}

func TestResolveForcedReturn(t *testing.T) {
	ctx := newCtxTenant(t)
	cur := field.NewBuilder(0, 0, _map.Id(103000800)).SetInstance(uuid.New()).Build()
	stub := &stubInfoProcessor{out: info.NewBuilder().SetForcedReturnMapId(_map.Id(103000890)).Build()}
	p := newProcessorWithInfo(logrus.New(), ctx, nil, stub)

	got, reason, err := p.Resolve(cur)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if reason != ReasonForcedReturn {
		t.Fatalf("reason = %s, want %s", reason, ReasonForcedReturn)
	}
	if got.MapId() != _map.Id(103000890) {
		t.Fatalf("MapId = %d, want 103000890", got.MapId())
	}
	if got.Instance() != uuid.Nil {
		t.Fatalf("Instance = %s, want Nil (relocation drops instance)", got.Instance())
	}
}

func TestResolveStayPut(t *testing.T) {
	ctx := newCtxTenant(t)
	inst := uuid.New()
	cur := field.NewBuilder(0, 0, _map.Id(100020000)).SetInstance(inst).Build()
	stub := &stubInfoProcessor{out: info.NewBuilder().SetForcedReturnMapId(_map.EmptyMapId).Build()}
	p := newProcessorWithInfo(logrus.New(), ctx, nil, stub)

	got, reason, err := p.Resolve(cur)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if reason != ReasonStayPut {
		t.Fatalf("reason = %s, want %s", reason, ReasonStayPut)
	}
	if got.MapId() != _map.Id(100020000) {
		t.Fatalf("MapId = %d, want 100020000", got.MapId())
	}
	if got.Instance() != inst {
		t.Fatalf("Instance = %s, want %s (stay put preserves instance)", got.Instance(), inst)
	}
}

func TestResolveInfoError(t *testing.T) {
	ctx := newCtxTenant(t)
	inst := uuid.New()
	cur := field.NewBuilder(0, 0, _map.Id(100020000)).SetInstance(inst).Build()
	stub := &stubInfoProcessor{err: errors.New("boom")}
	p := newProcessorWithInfo(logrus.New(), ctx, nil, stub)

	got, reason, err := p.Resolve(cur)
	if err != nil {
		t.Fatalf("Resolve must not error on info failure (degrades to stay put): %v", err)
	}
	if reason != ReasonStayPut {
		t.Fatalf("reason on info-error = %s, want stay_put", reason)
	}
	if got.MapId() != cur.MapId() || got.Instance() != cur.Instance() {
		t.Fatalf("on info-error must return current field unchanged")
	}
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	if err := Migration(db); err != nil {
		t.Fatalf("Migration: %v", err)
	}
	return db
}

func TestSetThenGetById(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)
	p := NewProcessor(logrus.New(), ctx, db)

	f := field.NewBuilder(0, 1, _map.Id(103000890)).SetInstance(uuid.Nil).Build()
	if _, err := p.Set(uint32(42), f); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := p.GetById(uint32(42))
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.MapId() != _map.Id(103000890) {
		t.Fatalf("MapId = %d, want 103000890", got.MapId())
	}
	if got.ChannelId() != 1 {
		t.Fatalf("ChannelId = %d, want 1", got.ChannelId())
	}
}

func TestGetByIdMissing(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)
	p := NewProcessor(logrus.New(), ctx, db)
	if _, err := p.GetById(uint32(999)); err == nil {
		t.Fatal("GetById on missing row should error (record not found)")
	}
}

func TestResolveAndSetForcedReturnPersists(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)
	stub := &stubInfoProcessor{out: info.NewBuilder().SetForcedReturnMapId(_map.Id(103000890)).Build()}
	p := newProcessorWithInfo(logrus.New(), ctx, db, stub)

	cur := field.NewBuilder(0, 0, _map.Id(103000800)).SetInstance(uuid.New()).Build()
	resolved, reason, err := p.Resolve(cur)
	if err != nil {
		t.Fatal(err)
	}
	if reason != ReasonForcedReturn {
		t.Fatalf("reason = %s", reason)
	}
	if _, err := p.Set(uint32(7), resolved); err != nil {
		t.Fatal(err)
	}

	got, err := p.GetById(uint32(7))
	if err != nil {
		t.Fatal(err)
	}
	if got.MapId() != _map.Id(103000890) {
		t.Fatalf("MapId = %d, want 103000890", got.MapId())
	}
	if got.Instance() != uuid.Nil {
		t.Fatalf("Instance must be Nil after relocation")
	}
}

func TestSetIsTenantScoped(t *testing.T) {
	db := newTestDB(t)
	tnA, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tnB, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctxA := tenant.WithContext(context.Background(), tnA)
	ctxB := tenant.WithContext(context.Background(), tnB)

	pA := NewProcessor(logrus.New(), ctxA, db)
	pB := NewProcessor(logrus.New(), ctxB, db)

	f := field.NewBuilder(0, 0, _map.Id(100020000)).SetInstance(uuid.Nil).Build()
	if _, err := pA.Set(uint32(7), f); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if _, err := pB.GetById(uint32(7)); err == nil {
		t.Fatal("tenant B must not see tenant A's row")
	}
}
