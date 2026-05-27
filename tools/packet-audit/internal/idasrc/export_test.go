package idasrc

import (
	"context"
	"testing"
)

func TestExportSourceResolve(t *testing.T) {
	src, err := NewExportSource("testdata/gms_v95_mini.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := src.Resolve(context.Background(), "CLogin::OnCheckPasswordResult")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(f.Calls) != 8 {
		t.Errorf("calls: got %d, want 8", len(f.Calls))
	}
	if f.Calls[7].Op != Decode2 {
		t.Errorf("calls[7]: got %v, want Decode2", f.Calls[7].Op)
	}
	if f.Direction != DirClientbound {
		t.Errorf("direction: got %v", f.Direction)
	}
}

func TestExportSourceDispatcherPerMob(t *testing.T) {
	src, err := NewExportSource("testdata/gms_v95_mini.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := src.Resolve(context.Background(), "CMob::OnDamaged")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	// JSON entry has 2 calls; "per-mob" dispatcher prepends 1 (Decode4 mobId).
	if len(f.Calls) != 3 {
		t.Fatalf("calls: got %d, want 3 (1 prefix + 2 leaf)", len(f.Calls))
	}
	if f.Calls[0].Op != Decode4 {
		t.Errorf("calls[0]: got %v, want Decode4 (dwMobId prefix)", f.Calls[0].Op)
	}
	if f.Calls[1].Op != Decode1 {
		t.Errorf("calls[1]: got %v, want Decode1 (damageType)", f.Calls[1].Op)
	}
	if f.Calls[2].Op != Decode4 {
		t.Errorf("calls[2]: got %v, want Decode4 (damage)", f.Calls[2].Op)
	}
}

func TestExportSourceDispatcherPerPet(t *testing.T) {
	src, err := NewExportSource("testdata/gms_v95_mini.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := src.Resolve(context.Background(), "CPet::OnAction")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	// "per-pet" prepends 2 (Decode4 characterId + Decode1 slot); 3 leaf calls.
	if len(f.Calls) != 5 {
		t.Fatalf("calls: got %d, want 5 (2 prefix + 3 leaf)", len(f.Calls))
	}
	if f.Calls[0].Op != Decode4 {
		t.Errorf("calls[0]: got %v, want Decode4 (characterId)", f.Calls[0].Op)
	}
	if f.Calls[1].Op != Decode1 {
		t.Errorf("calls[1]: got %v, want Decode1 (slot)", f.Calls[1].Op)
	}
	if f.Calls[2].Op != Decode1 {
		t.Errorf("calls[2]: got %v, want Decode1 (actionType)", f.Calls[2].Op)
	}
}

func TestExportSourceDispatcherPerPetRemote(t *testing.T) {
	src, err := NewExportSource("testdata/gms_v95_mini.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := src.Resolve(context.Background(), "CUserRemote::OnPetActivated")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	// "per-pet-remote" prepends only 1 (Decode4 characterId); 2 leaf calls.
	if len(f.Calls) != 3 {
		t.Fatalf("calls: got %d, want 3 (1 prefix + 2 leaf)", len(f.Calls))
	}
	if f.Calls[0].Op != Decode4 {
		t.Errorf("calls[0]: got %v, want Decode4 (characterId)", f.Calls[0].Op)
	}
	if f.Calls[1].Op != Decode1 {
		t.Errorf("calls[1]: got %v, want Decode1 (slot — leaf, not prefix)", f.Calls[1].Op)
	}
}

func TestExportSourceServerboundIgnoresDispatcherAnnotation(t *testing.T) {
	// The "CPet::DoAction" entry has no dispatcher annotation — its calls
	// pass through verbatim. Sanity-checks that serverbound entries (which
	// shouldn't carry a dispatcher prefix) round-trip correctly.
	src, err := NewExportSource("testdata/gms_v95_mini.json")
	if err != nil {
		t.Fatal(err)
	}
	f, err := src.Resolve(context.Background(), "CPet::DoAction")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(f.Calls) != 2 {
		t.Fatalf("calls: got %d, want 2 (no prefix, 2 leaf)", len(f.Calls))
	}
	if f.Direction != DirServerbound {
		t.Errorf("direction: got %v, want serverbound", f.Direction)
	}
}

func TestDispatcherPrefixUnknownKind(t *testing.T) {
	// Forward-compat: unrecognized kinds yield no prefix (warn-and-continue,
	// not error). A future dispatcher kind can be added without breaking
	// existing JSON entries that name it before its support lands.
	if p := dispatcherPrefix("per-something-new"); p != nil {
		t.Errorf("expected nil prefix for unknown kind; got %d entries", len(p))
	}
	if p := dispatcherPrefix(""); p != nil {
		t.Errorf("expected nil prefix for empty kind; got %d entries", len(p))
	}
}
