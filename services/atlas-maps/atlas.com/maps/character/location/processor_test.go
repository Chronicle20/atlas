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
