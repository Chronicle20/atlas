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
