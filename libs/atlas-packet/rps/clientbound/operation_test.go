package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// TestRPSGameOpen exercises the OPEN arm (mode 8) of the CRPSGameDlg::OnPacket
// dispatcher. Body = Decode4 int (ante). Mode byte is IDENTICAL across all
// five versions (docs/tasks/task-132-rps-npc-game/ida-rps-clientbound.md §0/§6
// — no per-version shift, unlike storage's jms -1 shift).
//
// packet-audit:verify packet=rps/clientbound/RpsOpen version=gms_v83 ida=0x7400ec
// packet-audit:verify packet=rps/clientbound/RpsOpen version=gms_v84 ida=0x761e10
// packet-audit:verify packet=rps/clientbound/RpsOpen version=gms_v87 ida=0x78acb0
// packet-audit:verify packet=rps/clientbound/RpsOpen version=gms_v95 ida=0x6d9e82
// packet-audit:verify packet=rps/clientbound/RpsOpen version=jms_v185 ida=0x7ae4d7
func TestRPSGameOpen(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewRPSGameOpen(8, 3000)
			b := input.Encode(l, ctx)(nil)
			// mode(1) + ante(4) = 5 bytes, no more.
			if len(b) != 5 {
				t.Fatalf("encoded length: got %d, want 5", len(b))
			}
			if b[0] != 8 {
				t.Errorf("mode: got %d, want 8", b[0])
			}
			ante := uint32(b[1]) | uint32(b[2])<<8 | uint32(b[3])<<16 | uint32(b[4])<<24
			if ante != 3000 {
				t.Errorf("ante: got %d, want 3000", ante)
			}

			output := Open{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Ante() != input.Ante() {
				t.Errorf("ante: got %v, want %v", output.Ante(), input.Ante())
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
// packet-audit:verify packet=rps/clientbound/RpsResult version=gms_v83 ida=0x740298
// packet-audit:verify packet=rps/clientbound/RpsResult version=gms_v84 ida=0x761fbc
// packet-audit:verify packet=rps/clientbound/RpsResult version=gms_v87 ida=0x78ae70
// packet-audit:verify packet=rps/clientbound/RpsResult version=gms_v95 ida=0x6d7372
// packet-audit:verify packet=rps/clientbound/RpsResult version=jms_v185 ida=0x7ae683
func TestRPSGameResult(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
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
// packet-audit:verify packet=rps/clientbound/RpsEnd version=gms_v83 ida=0x74009e
// packet-audit:verify packet=rps/clientbound/RpsEnd version=gms_v84 ida=0x761dc2
// packet-audit:verify packet=rps/clientbound/RpsEnd version=gms_v87 ida=0x78ac5a
// packet-audit:verify packet=rps/clientbound/RpsEnd version=gms_v95 ida=0x6d9ff0
// packet-audit:verify packet=rps/clientbound/RpsEnd version=jms_v185 ida=0x7ae489
func TestRPSGameEnd(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range pt.Variants {
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
