package discover

import (
	"testing"

	"github.com/Chronicle20/atlas/tools/packet-audit/internal/opregistry"
)

func TestReconcile(t *testing.T) {
	seeded := opregistry.NewVersionFile([]opregistry.Entry{
		{Op: "LOGIN_STATUS", Direction: opregistry.DirClientbound, Opcode: 0x000, FName: "CLogin::OnCheckPasswordResult", Provenance: "csv-import"},
		{Op: "GHOST_OP", Direction: opregistry.DirClientbound, Opcode: 0x0FF, FName: "CFoo::OnGhost", Provenance: "csv-import"},
	})
	discovered := []Discovered{
		{Opcode: 0x000, Handler: "CLogin::OnCheckPasswordResult", Address: "0x5e1230"}, // match
		{Opcode: 0x002, Handler: "CLogin::OnAccountInfoResult", Address: "0x5e2000"},   // new
		{Opcode: 0x0FF, Handler: "CBar::OnSomethingElse", Address: "0x5e3000"},         // collision
	}
	res := Reconcile(seeded, discovered, opregistry.DirClientbound)

	if len(res.Append) != 1 || res.Append[0].FName != "CLogin::OnAccountInfoResult" ||
		res.Append[0].Provenance != "ida-discovered" || res.Append[0].IDA == nil {
		t.Errorf("append = %+v", res.Append)
	}
	if len(res.MissingAtDiscovery) != 0 {
		// GHOST_OP's opcode WAS discovered (collision), so it's a collision, not missing
		t.Errorf("missing = %+v", res.MissingAtDiscovery)
	}
	if len(res.Collisions) != 1 || res.Collisions[0].Entry.Op != "GHOST_OP" {
		t.Errorf("collisions = %+v", res.Collisions)
	}
}

func TestReconcileMissing(t *testing.T) {
	seeded := opregistry.NewVersionFile([]opregistry.Entry{
		{Op: "NEVER_FOUND", Direction: opregistry.DirClientbound, Opcode: 0x0AA, FName: "CFoo::OnNever", Provenance: "csv-import"},
	})
	res := Reconcile(seeded, nil, opregistry.DirClientbound)
	if len(res.MissingAtDiscovery) != 1 {
		t.Errorf("missing = %+v (registry entries discovery can't find are flagged, never deleted)", res.MissingAtDiscovery)
	}
}

func TestOpNameFor(t *testing.T) {
	if got := opNameFor(Discovered{Opcode: 0x002}); got != "IDA_0X002" {
		t.Errorf("opNameFor(0x002) = %q, want IDA_0X002", got)
	}
	if got := opNameFor(Discovered{Opcode: 0x100}); got != "IDA_0X100" {
		t.Errorf("opNameFor(0x100) = %q, want IDA_0X100", got)
	}
}

func TestParseAddr(t *testing.T) {
	cases := []struct {
		in   string
		want uint64
	}{
		{"0x5e1230", 0x5e1230},
		{"0x5E1230", 0x5e1230},
		{"", 0},
		{"12345", 12345},
		{"bad", 0},
	}
	for _, tc := range cases {
		if got := parseAddr(tc.in); got != tc.want {
			t.Errorf("parseAddr(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestHasAlt(t *testing.T) {
	e := opregistry.Entry{FName: "Foo::OnBar", FNameAlts: []string{"Foo::OnBarV2"}}
	if !hasAlt(e, "Foo::OnBarV2") {
		t.Error("hasAlt should find FNameAlts entry")
	}
	if hasAlt(e, "NotThere") {
		t.Error("hasAlt false positive")
	}
}

// TestUnionDedupesSameOpcodeAndHandler verifies that two dispatchers reporting
// the same (opcode, handler) pair produce a single entry in the union.
func TestUnionDedupesSameOpcodeAndHandler(t *testing.T) {
	d := Discovered{Opcode: 0x10, Handler: "CField::OnEnterField", Address: "0xabc"}
	perDisp := []DispatcherResult{
		{Name: "CField::OnPacket", Addr: "0xf000", Cases: []Discovered{d}},
		{Name: "CWvsContext::OnPacket", Addr: "0xe000", Cases: []Discovered{d}},
	}
	cases, collisions := Union(perDisp)
	if len(collisions) != 0 {
		t.Errorf("expected 0 internal collisions, got %d: %+v", len(collisions), collisions)
	}
	if len(cases) != 1 {
		t.Errorf("expected 1 deduped case, got %d: %+v", len(cases), cases)
	}
	if cases[0].Opcode != 0x10 || cases[0].Handler != "CField::OnEnterField" {
		t.Errorf("unexpected case: %+v", cases[0])
	}
}

// TestUnionInternalCollision verifies that two dispatchers reporting the same
// opcode with different handlers produce an InternalCollision and NO case entry.
func TestUnionInternalCollision(t *testing.T) {
	dA := Discovered{Opcode: 0x20, Handler: "CField::OnSomething", Address: "0x1000"}
	dB := Discovered{Opcode: 0x20, Handler: "CLogin::OnSomethingElse", Address: "0x2000"}
	perDisp := []DispatcherResult{
		{Name: "CField::OnPacket", Addr: "0xf000", Cases: []Discovered{dA}},
		{Name: "CLogin::OnPacket", Addr: "0xe000", Cases: []Discovered{dB}},
	}
	cases, collisions := Union(perDisp)
	if len(collisions) != 1 {
		t.Errorf("expected 1 internal collision, got %d: %+v", len(collisions), collisions)
	}
	if len(cases) != 0 {
		t.Errorf("expected 0 cases (colliding op excluded), got %d: %+v", len(cases), cases)
	}
	col := collisions[0]
	if col.Opcode != 0x20 {
		t.Errorf("collision opcode = 0x%X, want 0x20", col.Opcode)
	}
	if col.HandlerA.Handler == col.HandlerB.Handler {
		t.Error("collision HandlerA and HandlerB must differ")
	}
}

// TestUnionMergesTwoDispatchers verifies that non-colliding ops from two
// dispatchers are all present in the union.
func TestUnionMergesTwoDispatchers(t *testing.T) {
	perDisp := []DispatcherResult{
		{
			Name: "CField::OnPacket",
			Addr: "0xf000",
			Cases: []Discovered{
				{Opcode: 0x10, Handler: "CField::OnEnterField", Address: "0x1000"},
				{Opcode: 0x11, Handler: "CField::OnLeaveField", Address: "0x1100"},
			},
		},
		{
			Name: "CLogin::OnPacket",
			Addr: "0xe000",
			Cases: []Discovered{
				{Opcode: 0x20, Handler: "CLogin::OnCheckPassword", Address: "0x2000"},
				{Opcode: 0x21, Handler: "CLogin::OnSelectWorld", Address: "0x2100"},
			},
		},
	}
	cases, collisions := Union(perDisp)
	if len(collisions) != 0 {
		t.Errorf("expected 0 collisions, got %d: %+v", len(collisions), collisions)
	}
	if len(cases) != 4 {
		t.Errorf("expected 4 cases in union, got %d: %+v", len(cases), cases)
	}
	// Verify sorted by opcode
	for i := 1; i < len(cases); i++ {
		if cases[i].Opcode <= cases[i-1].Opcode {
			t.Errorf("cases not sorted by opcode at index %d: 0x%X <= 0x%X", i, cases[i].Opcode, cases[i-1].Opcode)
		}
	}
}

// TestUnionZeroCaseDispatcherDoesNotBreakOthers verifies that a dispatcher
// yielding 0 cases does not prevent other dispatchers' cases from appearing in
// the union.
func TestUnionZeroCaseDispatcherDoesNotBreakOthers(t *testing.T) {
	perDisp := []DispatcherResult{
		{Name: "CField::OnPacket", Addr: "0xf000", Cases: []Discovered{}}, // 0 cases
		{
			Name: "CLogin::OnPacket",
			Addr: "0xe000",
			Cases: []Discovered{
				{Opcode: 0x30, Handler: "CLogin::OnFoo", Address: "0x3000"},
			},
		},
	}
	cases, collisions := Union(perDisp)
	if len(collisions) != 0 {
		t.Errorf("expected 0 collisions, got %d", len(collisions))
	}
	if len(cases) != 1 || cases[0].Opcode != 0x30 {
		t.Errorf("expected 1 case from the non-zero dispatcher, got %+v", cases)
	}
}
