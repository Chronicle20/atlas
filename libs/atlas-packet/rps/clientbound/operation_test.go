package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// rpsVariants extends the shared pt.Variants set with the four legacy GMS
// versions this fixture also verifies: gms_v48, gms_v61, gms_v72, gms_v79.
// RPS_GAME's CRPSGameDlg::OnPacket dispatcher body is version-invariant — a
// live IDA re-audit (docs/tasks/task-132-rps-npc-game/ida-rps-legacy-reaudit.md)
// proved v48 (@0x5ADB94), v61 (@0x63BF0E), v72 (@0x69c54b) and v79
// (@0x6c1d5b) are byte-identical to the already-verified v83 dispatcher
// (@0x73fff1): same mode bytes (8=OPEN, 9-12 delegate to the RESULT
// sub-dispatcher, 13=END) and identical per-arm reads; only the RPS_GAME
// opcode shifts per version (v48=237/0xED, v61=242/0xF2, v72=278/0x116,
// v79=290/0x122, v83=312/0x138). NOTE: the v48/v61 IDBs mislabel this
// dispatcher (the export previously carried the WRONG function under the
// CRPSGameDlg::OnPacket key — a channel/find-player dialog for v48, the
// trunk dialog for v61); the export was surgically corrected with the
// real dispatcher's OPEN/RESULT/END arms before this cell could be pinned.
// The codec in operation.go carries no MajorVersion gate, so these four
// versions are appended to a local copy of pt.Variants rather than the
// shared global slice (unrelated packets whose codecs DO version-gate must
// not be exercised against untested legacy versions as a side effect of
// this change).
var rpsVariants = append(append([]pt.TenantVariant{}, pt.Variants...),
	pt.TenantVariant{Name: "GMS v48", Region: "GMS", MajorVersion: 48, MinorVersion: 1},
	pt.TenantVariant{Name: "GMS v61", Region: "GMS", MajorVersion: 61, MinorVersion: 1},
	pt.TenantVariant{Name: "GMS v72", Region: "GMS", MajorVersion: 72, MinorVersion: 1},
	pt.TenantVariant{Name: "GMS v79", Region: "GMS", MajorVersion: 79, MinorVersion: 1},
)

// TestRPSGameOpen exercises the OPEN arm (mode 8) of the CRPSGameDlg::OnPacket
// dispatcher. Body = Decode4 int = the NPC template id (the client loads
// Npc/{id}.img for the dealer's face; NOT the ante). Mode byte is IDENTICAL
// across all seven versions (docs/tasks/task-132-rps-npc-game/ida-rps-clientbound.md
// §0/§6 — no per-version shift, unlike storage's jms -1 shift).
//
// packet-audit:verify packet=rps/clientbound/RpsOpen version=gms_v48 ida=0x5adc8f
// packet-audit:verify packet=rps/clientbound/RpsOpen version=gms_v61 ida=0x63c009
// packet-audit:verify packet=rps/clientbound/RpsOpen version=gms_v72 ida=0x69c646
// packet-audit:verify packet=rps/clientbound/RpsOpen version=gms_v79 ida=0x6c1e56
// packet-audit:verify packet=rps/clientbound/RpsOpen version=gms_v83 ida=0x7400ec
// packet-audit:verify packet=rps/clientbound/RpsOpen version=gms_v84 ida=0x761e10
// packet-audit:verify packet=rps/clientbound/RpsOpen version=gms_v87 ida=0x78acb0
// packet-audit:verify packet=rps/clientbound/RpsOpen version=gms_v95 ida=0x6d9e82
// packet-audit:verify packet=rps/clientbound/RpsOpen version=jms_v185 ida=0x7ae4d7
func TestRPSGameOpen(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range rpsVariants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewRPSGameOpen(8, 9000019)
			b := input.Encode(l, ctx)(nil)
			// mode(1) + npcId(4) = 5 bytes, no more.
			if len(b) != 5 {
				t.Fatalf("encoded length: got %d, want 5", len(b))
			}
			if b[0] != 8 {
				t.Errorf("mode: got %d, want 8", b[0])
			}
			npcId := uint32(b[1]) | uint32(b[2])<<8 | uint32(b[3])<<16 | uint32(b[4])<<24
			if npcId != 9000019 {
				t.Errorf("npcId: got %d, want 9000019", npcId)
			}

			output := Open{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.NpcId() != input.NpcId() {
				t.Errorf("npcId: got %v, want %v", output.NpcId(), input.NpcId())
			}
		})
	}
}

// TestRPSGameStartSelect exercises the START_SELECT arm (mode 9): mode byte
// only, no body (client enables R/P/S buttons + arms the selection timer, no
// further wire reads — §0/§1-§5 of the IDA note). The mode byte is version-
// invariant, so the fixture runs across all seven versions and proves the wire
// format everywhere.
//
// NOTE — matrix promotion pending. This cell has no packet-audit:verify marker
// yet: promoting it to ✅ requires splicing a CRPSGameDlg::OnPacket#START_SELECT
// arm (the verbatim case-9 decompile) into each version's IDA export, which
// needs a live IDA pass (docs/tasks/task-132-rps-npc-game — the mode-9 arm was
// out of scope for the original Task-14 export splice). The case-9 handler
// addresses are known from the committed clientbound note §1-§5 (v83 0x7402e9,
// v84 0x76200d, v87 0x78aec1, v95 0x6d72ec, jms185 0x7ae6d4); the four legacy
// versions share the same byte-identical dispatcher but their case-9 offsets
// were never derived. Until the export-splice pass runs, the byte-fixture below
// is the wire-format guarantee; no fabricated address is cited.
func TestRPSGameStartSelect(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range rpsVariants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewRPSGameStartSelect(9)
			b := input.Encode(l, ctx)(nil)
			if len(b) != 1 || b[0] != 9 {
				t.Fatalf("StartSelect body: got %v, want [9]", b)
			}
			output := StartSelect{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// TestRPSGameResult exercises the RESULT arm (mode 11): Decode1 npcThrow +
// Decode1 straightVictoryCount (SIGNED int8). Uses a NEGATIVE
// straightVictoryCount (-5) to prove WriteInt8/ReadInt8 sign-extension is
// correct — the client branches `if (v < 0)` on this exact field (§1-§5 of
// the IDA note), so a naive unsigned round-trip would silently corrupt the
// game-over signal.
//
// packet-audit:verify packet=rps/clientbound/RpsResult version=gms_v48 ida=0x5ade39
// packet-audit:verify packet=rps/clientbound/RpsResult version=gms_v61 ida=0x63c1b3
// packet-audit:verify packet=rps/clientbound/RpsResult version=gms_v72 ida=0x69c7f2
// packet-audit:verify packet=rps/clientbound/RpsResult version=gms_v79 ida=0x6c2002
// packet-audit:verify packet=rps/clientbound/RpsResult version=gms_v83 ida=0x740298
// packet-audit:verify packet=rps/clientbound/RpsResult version=gms_v84 ida=0x761fbc
// packet-audit:verify packet=rps/clientbound/RpsResult version=gms_v87 ida=0x78ae70
// packet-audit:verify packet=rps/clientbound/RpsResult version=gms_v95 ida=0x6d7372
// packet-audit:verify packet=rps/clientbound/RpsResult version=jms_v185 ida=0x7ae683
func TestRPSGameResult(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range rpsVariants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewRPSGameResult(11, 2, -5)
			b := input.Encode(l, ctx)(nil)
			// mode(1) + npcThrow(1) + straightVictoryCount(1) = 3 bytes, no more.
			if len(b) != 3 {
				t.Fatalf("encoded length: got %d, want 3", len(b))
			}
			if b[0] != 11 {
				t.Errorf("mode: got %d, want 11", b[0])
			}
			if b[1] != 2 {
				t.Errorf("npcThrow: got %d, want 2", b[1])
			}
			if int8(b[2]) != -5 {
				t.Errorf("straightVictoryCount byte: got %d, want -5 (as int8)", int8(b[2]))
			}

			output := Result{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.NpcThrow() != input.NpcThrow() {
				t.Errorf("npcThrow: got %v, want %v", output.NpcThrow(), input.NpcThrow())
			}
			if output.StraightVictoryCount() != -5 {
				t.Errorf("straightVictoryCount: got %v, want -5", output.StraightVictoryCount())
			}
			if output.StraightVictoryCount() != input.StraightVictoryCount() {
				t.Errorf("straightVictoryCount round-trip: got %v, want %v", output.StraightVictoryCount(), input.StraightVictoryCount())
			}
		})
	}
}

// TestRPSGameEnd exercises the CLOSE arm (mode 13): mode byte only, no body
// (CWnd::Destroy, no further wire reads — §1-§5 of the IDA note).
//
// packet-audit:verify packet=rps/clientbound/RpsEnd version=gms_v48 ida=0x5adc41
// packet-audit:verify packet=rps/clientbound/RpsEnd version=gms_v61 ida=0x63bfbb
// packet-audit:verify packet=rps/clientbound/RpsEnd version=gms_v72 ida=0x69c5f8
// packet-audit:verify packet=rps/clientbound/RpsEnd version=gms_v79 ida=0x6c1e08
// packet-audit:verify packet=rps/clientbound/RpsEnd version=gms_v83 ida=0x74009e
// packet-audit:verify packet=rps/clientbound/RpsEnd version=gms_v84 ida=0x761dc2
// packet-audit:verify packet=rps/clientbound/RpsEnd version=gms_v87 ida=0x78ac5a
// packet-audit:verify packet=rps/clientbound/RpsEnd version=gms_v95 ida=0x6d9ff0
// packet-audit:verify packet=rps/clientbound/RpsEnd version=jms_v185 ida=0x7ae489
func TestRPSGameEnd(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range rpsVariants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewRPSGameEnd(13)
			b := input.Encode(l, ctx)(nil)
			if len(b) != 1 || b[0] != 13 {
				t.Fatalf("End body: got %v, want [13]", b)
			}
			output := End{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}
