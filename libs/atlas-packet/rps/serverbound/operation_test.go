package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// TestOperationRoundTrip exercises the generic bodyless-arm decode (mode byte
// only). It represents all five bodyless RPS_ACTION senders — OnBtStart(0),
// Update/timeout(2), OnBtContinue(3), OnBtExit(4), OnBtRetry(5) — whose ENTIRE
// wire content, per docs/tasks/task-132-rps-npc-game/ida-rps-serverbound.md
// §0/§1-§5, is the sub-op byte alone (no further fields). Registry primary
// fname is CRPSGameDlg::OnBtStart for gms_v83/v87/v95/jms_v185 and
// CRPSGameDlg::Update for gms_v84 (docs/packets/registry/gms_v84.yaml,
// task-100 cluster-H); both addresses are cited below since the Atlas struct
// is identical either way (1-byte mode decode).
//
// packet-audit:verify packet=rps/serverbound/RpsOperation version=gms_v83 ida=0x7403d0
// packet-audit:verify packet=rps/serverbound/RpsOperation version=gms_v84 ida=0x760e64
// packet-audit:verify packet=rps/serverbound/RpsOperation version=gms_v87 ida=0x78afa8
// packet-audit:verify packet=rps/serverbound/RpsOperation version=gms_v95 ida=0x6d6860
// packet-audit:verify packet=rps/serverbound/RpsOperation version=jms_v185 ida=0x7ae7bb
func TestOperationRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Operation{mode: 5}
			output := Operation{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

// TestOperationRoundTripBodylessArms fixtures EACH bodyless sub-op value
// individually (0=START, 2=TIMEOUT, 3=CONTINUE, 4=EXIT, 5=RETRY) — every one
// is a real, distinct, IDA-decompiled sender (OnBtStart/Update/OnBtContinue/
// OnBtExit/OnBtRetry respectively; see the IDA note §1-§5 per-version tables)
// and every one's full body is the bare sub-op byte, so a single generic
// Operation{mode} decode legitimately represents all five — this is not a
// mode-byte-only STUB (AP-7): there is no further body to omit.
func TestOperationRoundTripBodylessArms(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	cases := []struct {
		name string
		mode byte
	}{
		{"START", 0},
		{"TIMEOUT", 2},
		{"CONTINUE", 3},
		{"EXIT", 4},
		{"RETRY", 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, v := range pt.Variants {
				t.Run(v.Name, func(t *testing.T) {
					ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
					input := Operation{mode: c.mode}
					b := input.Encode(l, ctx)(nil)
					if len(b) != 1 {
						t.Fatalf("encoded length: got %d, want 1", len(b))
					}
					if b[0] != c.mode {
						t.Errorf("mode byte: got %d, want %d", b[0], c.mode)
					}
					output := Operation{}
					pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
					if output.Mode() != c.mode {
						t.Errorf("mode: got %v, want %v", output.Mode(), c.mode)
					}
				})
			}
		})
	}
}

// TestOperationSelectRoundTrip exercises the SELECT arm's body: a single RAW
// throw byte (0=Rock/1=Paper/2=Scissors), per CRPSGameDlg::SendSelection
// (Encode1(1) mode + Encode1(throw), IDA note §1-§5). The mode byte itself is
// captured by Operation (see TestOperationRoundTrip); this struct decodes
// only the trailing throw byte, mirroring storage's OperationMeso/
// OperationStoreAsset/OperationRetrieveAsset convention (wrapper mode + a
// separate body-only struct).
//
// packet-audit:verify packet=rps/serverbound/RpsOperationSelect version=gms_v83 ida=0x7405a0
// packet-audit:verify packet=rps/serverbound/RpsOperationSelect version=gms_v84 ida=0x7622c4
// packet-audit:verify packet=rps/serverbound/RpsOperationSelect version=gms_v87 ida=0x78b178
// packet-audit:verify packet=rps/serverbound/RpsOperationSelect version=gms_v95 ida=0x6d6ae0
// packet-audit:verify packet=rps/serverbound/RpsOperationSelect version=jms_v185 ida=0x7ae98b
func TestOperationSelectRoundTrip(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, throw := range []byte{0, 1, 2} {
		t.Run(throwName(throw), func(t *testing.T) {
			for _, v := range pt.Variants {
				t.Run(v.Name, func(t *testing.T) {
					ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
					input := OperationSelect{throw: throw}
					b := input.Encode(l, ctx)(nil)
					if len(b) != 1 {
						t.Fatalf("encoded length: got %d, want 1", len(b))
					}
					if b[0] != throw {
						t.Errorf("throw byte: got %d, want %d", b[0], throw)
					}
					output := OperationSelect{}
					pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
					if output.Throw() != throw {
						t.Errorf("throw: got %v, want %v", output.Throw(), throw)
					}
				})
			}
		})
	}
}

func throwName(b byte) string {
	switch b {
	case 0:
		return "Rock"
	case 1:
		return "Paper"
	case 2:
		return "Scissors"
	default:
		return "Unknown"
	}
}
