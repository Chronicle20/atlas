package controller

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testProcessor(t *testing.T, r *Registry, ten tenant.Model, live []uint32, hidden map[uint32]bool) *ProcessorImpl {
	t.Helper()
	registry = r // package-level singleton for the test; restore after
	t.Cleanup(func() { registry = nil })
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, tenant.WithContext(context.Background(), ten)).(*ProcessorImpl)
	p.fieldIdsFn = func(field.Model) ([]uint32, error) { return live, nil }
	p.hiddenFn = func(id uint32) bool { return hidden[id] }
	return p
}

func TestTryClaimClaimsUnclaimed(t *testing.T) {
	r, ten, f := setupRegistry(t)
	p := testProcessor(t, r, ten, []uint32{7}, nil)
	won, err := p.TryClaim(f, 1000, 7)
	if err != nil || !won {
		t.Fatalf("expected claim: won=%v err=%v", won, err)
	}
}

func TestTryClaimRespectsLiveController(t *testing.T) {
	r, ten, f := setupRegistry(t)
	_, _ = r.Claim(context.Background(), ten, f, 1000, 7)
	p := testProcessor(t, r, ten, []uint32{7, 8}, nil)
	won, err := p.TryClaim(f, 1000, 8)
	if err != nil || won {
		t.Fatalf("live controller must be kept: won=%v err=%v", won, err)
	}
}

func TestTryClaimReissuesForCurrentController(t *testing.T) {
	r, ten, f := setupRegistry(t)
	_, _ = r.Claim(context.Background(), ten, f, 1000, 7)
	p := testProcessor(t, r, ten, []uint32{7}, nil)
	won, err := p.TryClaim(f, 1000, 7)
	if err != nil || !won {
		t.Fatalf("current controller must get a re-issue: won=%v err=%v", won, err)
	}
}

func TestTryClaimReplacesStaleController(t *testing.T) {
	r, ten, f := setupRegistry(t)
	_, _ = r.Claim(context.Background(), ten, f, 1000, 99) // 99 not live
	p := testProcessor(t, r, ten, []uint32{7}, nil)
	won, err := p.TryClaim(f, 1000, 7)
	if err != nil || !won {
		t.Fatalf("stale entry must be re-claimed: won=%v err=%v", won, err)
	}
	cur, _, _ := r.ControllerOf(context.Background(), ten, f, 1000)
	if cur != 7 {
		t.Fatalf("expected new controller 7, got %d", cur)
	}
}

func TestTryClaimHiddenClaimsNothing(t *testing.T) {
	r, ten, f := setupRegistry(t)
	p := testProcessor(t, r, ten, []uint32{7}, map[uint32]bool{7: true})
	won, err := p.TryClaim(f, 1000, 7)
	if err != nil || won {
		t.Fatalf("hidden character must not claim: won=%v err=%v", won, err)
	}
}

func TestReleaseForReturnsReleasedIds(t *testing.T) {
	r, ten, f := setupRegistry(t)
	ctx := context.Background()
	_, _ = r.Claim(ctx, ten, f, 1000, 7)
	_, _ = r.Claim(ctx, ten, f, 1001, 7)
	_, _ = r.Claim(ctx, ten, f, 1002, 8)
	p := testProcessor(t, r, ten, []uint32{7, 8}, nil)
	released, err := p.ReleaseFor(f, 7)
	if err != nil || len(released) != 2 {
		t.Fatalf("expected 2 released, got %v err %v", released, err)
	}
	all, _ := r.GetAll(ctx, ten, f)
	if len(all) != 1 {
		t.Fatalf("only 1002 should remain, got %v", all)
	}
}

func TestElectForLeastLoadedSkipsHiddenAndExcluded(t *testing.T) {
	r, ten, f := setupRegistry(t)
	ctx := context.Background()
	// 8 already controls one NPC; 9 is hidden; 10 exiting (excluded); 11 free.
	_, _ = r.Claim(ctx, ten, f, 2000, 8)
	p := testProcessor(t, r, ten, []uint32{8, 9, 10, 11}, map[uint32]bool{9: true})
	got, err := p.ElectFor(f, []uint32{1000, 1001}, 10)
	if err != nil {
		t.Fatalf("ElectFor: %v", err)
	}
	// 11 is least-loaded (0 vs 8's 1) and visible: first NPC -> 11; then
	// counts tie 1-1, either 8 or 11 wins the second — assert both NPCs got
	// a visible, non-excluded winner.
	for npc, winner := range got {
		if winner == 9 || winner == 10 {
			t.Fatalf("npc %d assigned to hidden/excluded %d", npc, winner)
		}
	}
	if len(got) != 2 {
		t.Fatalf("both NPCs must be assigned, got %v", got)
	}
	if got[1000] != 11 && got[1001] != 11 {
		t.Fatalf("least-loaded 11 must win at least one NPC, got %v", got)
	}
}

func TestElectForNoCandidatesLeavesUncontrolled(t *testing.T) {
	r, ten, f := setupRegistry(t)
	p := testProcessor(t, r, ten, []uint32{9}, map[uint32]bool{9: true})
	got, err := p.ElectFor(f, []uint32{1000})
	if err != nil || len(got) != 0 {
		t.Fatalf("only-hidden field must elect nobody: %v err %v", got, err)
	}
	_, ok, _ := r.ControllerOf(context.Background(), ten, f, 1000)
	if ok {
		t.Fatalf("npc must stay uncontrolled")
	}
}

func TestUncontrolledIn(t *testing.T) {
	r, ten, f := setupRegistry(t)
	ctx := context.Background()
	_, _ = r.Claim(ctx, ten, f, 1000, 7)  // live
	_, _ = r.Claim(ctx, ten, f, 1001, 99) // stale
	p := testProcessor(t, r, ten, []uint32{7}, nil)
	unc, err := p.UncontrolledIn(f, []uint32{1000, 1001, 1002})
	if err != nil {
		t.Fatalf("UncontrolledIn: %v", err)
	}
	want := map[uint32]bool{1001: true, 1002: true}
	if len(unc) != 2 || !want[unc[0]] || !want[unc[1]] {
		t.Fatalf("expected {1001,1002}, got %v", unc)
	}
}

func TestIsControllerFailOpen(t *testing.T) {
	// nil registry (pre-init) must fail open to true.
	registry = nil
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	f := field.NewBuilder(0, 1, 100000000).Build()
	if !IsController(context.Background(), ten, f, 7, 1000) {
		t.Fatalf("nil registry must fail open")
	}
	// uncontrolled NPC -> not controller
	r, ten2, f2 := setupRegistry(t)
	registry = r
	t.Cleanup(func() { registry = nil })
	if IsController(context.Background(), ten2, f2, 7, 1000) {
		t.Fatalf("uncontrolled NPC must not pass the controller guard")
	}
	_, _ = r.Claim(context.Background(), ten2, f2, 1000, 7)
	if !IsController(context.Background(), ten2, f2, 7, 1000) {
		t.Fatalf("recorded controller must pass")
	}
	if IsController(context.Background(), ten2, f2, 8, 1000) {
		t.Fatalf("non-controller must not pass")
	}
}
