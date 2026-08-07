package hidden

import (
	"context"
	"errors"
	"testing"
	"time"

	buff "atlas-monsters/character/buff"

	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestReconciliationRemovesStaleKeepsActive(t *testing.T) {
	r := setup(t) // from registry_test.go
	ctx := context.Background()
	ten := testTenant(t)
	_ = r.Add(ctx, ten, 1) // stale: no hide buff upstream
	_ = r.Add(ctx, ten, 2) // active: hide buff upstream
	_ = r.Add(ctx, ten, 3) // fetch error: must be kept (fail-safe)

	l, _ := test.NewNullLogger()
	task := NewReconciliationTask(l, ctx, time.Minute)
	task.registry = r
	task.buffsFn = func(_ tenant.Model, characterId uint32) ([]buff.Model, error) {
		switch characterId {
		case 2:
			return []buff.Model{buff.NewModel(9101004, time.Now().Add(time.Hour))}, nil
		case 3:
			return nil, errors.New("buffs unavailable")
		default:
			return []buff.Model{}, nil
		}
	}
	task.Run()

	ms, _ := r.MemberSet(ctx, ten)
	if _, ok := ms[1]; ok {
		t.Fatalf("stale member 1 must be removed")
	}
	if _, ok := ms[2]; !ok {
		t.Fatalf("active member 2 must be kept")
	}
	if _, ok := ms[3]; !ok {
		t.Fatalf("member 3 must be kept when the buffs fetch fails")
	}
}
