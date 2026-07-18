package serverbound

import (
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// rpsServerVariants extends the shared pt.Variants set with the four legacy
// GMS versions this fixture also verifies: gms_v48, gms_v61, gms_v72,
// gms_v79. A live IDA re-audit
// (docs/tasks/task-132-rps-npc-game/ida-rps-legacy-reaudit.md) proved the
// RPS_ACTION serverbound sender set (OnBtStart/OnBtContinue/OnBtExit/
// OnBtRetry/Update/SendSelection, 6-helper send set with a leading sub-op
// byte) is byte-identical in body across all four legacy versions and the
// already-verified v83 baseline — only the RPS_ACTION opcode shifts per
// version (v48=111/0x6F, v61=124/0x7C, v72=134/0x86, v79=133/0x85,
// v83=136/0x88). The codec in operation.go/operation_select.go carries no
// MajorVersion gate, so these four versions are appended to a local copy of
// pt.Variants rather than the shared global slice (unrelated packets whose
// codecs DO version-gate must not be exercised against untested legacy
// versions as a side effect of this change).
var rpsServerVariants = append(append([]pt.TenantVariant{}, pt.Variants...),
	pt.TenantVariant{Name: "GMS v48", Region: "GMS", MajorVersion: 48, MinorVersion: 1},
	pt.TenantVariant{Name: "GMS v61", Region: "GMS", MajorVersion: 61, MinorVersion: 1},
	pt.TenantVariant{Name: "GMS v72", Region: "GMS", MajorVersion: 72, MinorVersion: 1},
	pt.TenantVariant{Name: "GMS v79", Region: "GMS", MajorVersion: 79, MinorVersion: 1},
)

// TestOperationRoundTrip exercises the generic bodyless-arm decode (mode byte
// only). It represents all five bodyless RPS_ACTION senders — OnBtStart(0),
// Update/timeout(2), OnBtContinue(3), OnBtExit(4), OnBtRetry(5) — whose ENTIRE
// wire content, per docs/tasks/task-132-rps-npc-game/ida-rps-serverbound.md
// §0/§1-§5, is the sub-op byte alone (no further fields). Registry primary
// fname is CRPSGameDlg::OnBtStart for gms_v48/v61/v72/v79/v83/v87/v95/jms_v185
// and CRPSGameDlg::Update for gms_v84 (docs/packets/registry/gms_v84.yaml,
// task-100 cluster-H); both addresses are cited below since the Atlas struct
// is identical either way (1-byte mode decode). v48/v61/v72/v79 addresses are
// the field-specific Encode1(0) call site inside each version's OnBtStart
// sender, live-IDA-decompiled 2026-07-16 (v48 port 13337, v61 port 13338,
// v72 port 13339, v79 port 13340) — see ida-rps-legacy-reaudit.md.
//
// packet-audit:verify packet=rps/serverbound/RpsOperation version=gms_v48 ida=0x5adf94
// packet-audit:verify packet=rps/serverbound/RpsOperation version=gms_v61 ida=0x63c30e
// packet-audit:verify packet=rps/serverbound/RpsOperation version=gms_v72 ida=0x69c950
// packet-audit:verify packet=rps/serverbound/RpsOperation version=gms_v79 ida=0x6c2160
// packet-audit:verify packet=rps/serverbound/RpsOperation version=gms_v83 ida=0x7403d0
// packet-audit:verify packet=rps/serverbound/RpsOperation version=gms_v84 ida=0x760e64
// packet-audit:verify packet=rps/serverbound/RpsOperation version=gms_v87 ida=0x78afa8
// packet-audit:verify packet=rps/serverbound/RpsOperation version=gms_v95 ida=0x6d6860
// packet-audit:verify packet=rps/serverbound/RpsOperation version=jms_v185 ida=0x7ae7bb
func TestOperationRoundTrip(t *testing.T) {
	for _, v := range rpsServerVariants {
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
			for _, v := range rpsServerVariants {
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
// packet-audit:verify packet=rps/serverbound/RpsOperationSelect version=gms_v48 ida=0x5ae16c
// packet-audit:verify packet=rps/serverbound/RpsOperationSelect version=gms_v61 ida=0x63c4e6
// packet-audit:verify packet=rps/serverbound/RpsOperationSelect version=gms_v72 ida=0x69cb2d
// packet-audit:verify packet=rps/serverbound/RpsOperationSelect version=gms_v79 ida=0x6c233d
// packet-audit:verify packet=rps/serverbound/RpsOperationSelect version=gms_v83 ida=0x7405a0
// packet-audit:verify packet=rps/serverbound/RpsOperationSelect version=gms_v84 ida=0x7622c4
// packet-audit:verify packet=rps/serverbound/RpsOperationSelect version=gms_v87 ida=0x78b178
// packet-audit:verify packet=rps/serverbound/RpsOperationSelect version=gms_v95 ida=0x6d6ae0
// packet-audit:verify packet=rps/serverbound/RpsOperationSelect version=jms_v185 ida=0x7ae98b
func TestOperationSelectRoundTrip(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, throw := range []byte{0, 1, 2} {
		t.Run(throwName(throw), func(t *testing.T) {
			for _, v := range rpsServerVariants {
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
