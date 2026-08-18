package location

import (
	"atlas-maps/data/map/info"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

// TestSetState_TransitionsWithoutDisturbingPosition proves the state column is
// an independent discriminator: flipping it must not move the character.
func TestSetState_TransitionsWithoutDisturbingPosition(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)

	const characterId uint32 = 42
	f := field.NewBuilder(world.Id(1), 7, _map.Id(100000000)).SetInstance(uuid.Nil).Build()

	p := NewProcessor(logrus.New(), ctx, db)
	if _, err := p.Set(characterId, f); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// A freshly written row is OFFLINE until something asserts liveness.
	m, err := p.GetById(characterId)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if m.State() != characterconst.PresenceStateOffline {
		t.Errorf("initial state = %q, want OFFLINE", m.State())
	}

	if err := p.SetState(characterId, characterconst.PresenceStateInCashShop); err != nil {
		t.Fatalf("SetState: %v", err)
	}

	m, err = p.GetById(characterId)
	if err != nil {
		t.Fatalf("GetById after SetState: %v", err)
	}
	if m.State() != characterconst.PresenceStateInCashShop {
		t.Errorf("state = %q, want IN_CASH_SHOP", m.State())
	}
	if m.WorldId() != world.Id(1) || m.ChannelId() != 7 || m.MapId() != _map.Id(100000000) {
		t.Errorf("SetState disturbed position: world=%d channel=%d map=%d", m.WorldId(), m.ChannelId(), m.MapId())
	}
}

// TestSet_PreservesState proves a position write does not reset the
// discriminator. LOGOUT calls Set then SetState; CHANGE_MAP calls Set alone and
// must leave liveness as it found it. upsertLocation used db.Save (a full-row
// overwrite), which would silently zero the state here.
func TestSet_PreservesState(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)

	const characterId uint32 = 43
	f := field.NewBuilder(world.Id(1), 2, _map.Id(100000000)).SetInstance(uuid.Nil).Build()

	p := NewProcessor(logrus.New(), ctx, db)
	if _, err := p.Set(characterId, f); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := p.SetState(characterId, characterconst.PresenceStateInField); err != nil {
		t.Fatalf("SetState: %v", err)
	}

	moved := field.NewBuilder(world.Id(1), 2, _map.Id(104000000)).SetInstance(uuid.Nil).Build()
	if _, err := p.Set(characterId, moved); err != nil {
		t.Fatalf("Set after move: %v", err)
	}

	m, err := p.GetById(characterId)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if m.MapId() != _map.Id(104000000) {
		t.Errorf("map = %d, want 104000000", m.MapId())
	}
	if m.State() != characterconst.PresenceStateInField {
		t.Errorf("Set reset the state to %q, want IN_FIELD preserved", m.State())
	}
}

// TestSetStateIfOnline_DoesNotResurrectOfflineRow is the ordering rule of
// design §1.3: a CHARACTER_EXIT that arrives after LOGOUT must not flip a
// logged-off character back to IN_FIELD.
func TestSetStateIfOnline_DoesNotResurrectOfflineRow(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)

	const characterId uint32 = 45
	f := field.NewBuilder(world.Id(1), 3, _map.Id(100000000)).SetInstance(uuid.Nil).Build()

	p := NewProcessor(logrus.New(), ctx, db)
	if _, err := p.Set(characterId, f); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := p.SetState(characterId, characterconst.PresenceStateOffline); err != nil {
		t.Fatalf("SetState OFFLINE: %v", err)
	}

	// The late CHARACTER_EXIT.
	if err := p.SetStateIfOnline(characterId, characterconst.PresenceStateInField); err != nil {
		t.Fatalf("SetStateIfOnline: %v", err)
	}

	m, err := p.GetById(characterId)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if m.State() != characterconst.PresenceStateOffline {
		t.Errorf("late CHARACTER_EXIT resurrected the row to %q, want OFFLINE", m.State())
	}
}

// TestSetStateIfOnline_AppliesWhenOnline is the same call on a live row.
func TestSetStateIfOnline_AppliesWhenOnline(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)

	const characterId uint32 = 46
	f := field.NewBuilder(world.Id(1), 3, _map.Id(100000000)).SetInstance(uuid.Nil).Build()

	p := NewProcessor(logrus.New(), ctx, db)
	if _, err := p.Set(characterId, f); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := p.SetState(characterId, characterconst.PresenceStateInCashShop); err != nil {
		t.Fatalf("SetState: %v", err)
	}
	if err := p.SetStateIfOnline(characterId, characterconst.PresenceStateInField); err != nil {
		t.Fatalf("SetStateIfOnline: %v", err)
	}

	m, err := p.GetById(characterId)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if m.State() != characterconst.PresenceStateInField {
		t.Errorf("state = %q, want IN_FIELD", m.State())
	}
}
