package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// MTS_CHARGE_PARAM_RESULT (CITC::OnChargeParamResult) — the bodiless "charge
// parameter result" the client expects after ITC_STATUS_CHARGE (the MTS "Charge"
// button). Every implemented client's CITC::OnChargeParamResult handler was
// decompiled and reads NOTHING from the CInPacket: it clears the ITC request
// latch (this[6]=0 / m_bITCRequestSent=0), looks up the billing web URL from the
// StringPool, and opens it via open_web_site. The opcode alone is the signal, so
// the wire body is empty — version-stable across every version it exists in, so
// the codec needs no version gating.
//
// The handler read-site (function entry) addresses below are IDA-verified from
// each client's IDB (GMS_v9 IDBs). They are pinned as packet-audit:verify machine
// markers on TestMtsChargeParamResultGolden; each carries a fresh evidence record
// (docs/packets/evidence/<version>/field.clientbound.MtsChargeParamResult.yaml)
// keyed to the export function CITC::OnChargeParamResult, which promotes
// MTS_CHARGE_PARAM_RESULT to ✅ in the coverage matrix for every version it is
// implemented in (task-113). NOT jms_v185: the jms tenant template does not wire
// the MtsChargeParamResult writer, so the packet is not implemented there.
//
//	version   CITC::OnChargeParamResult (handler entry)   dispatch opcode
//	gms_v61   0x52d691                                    0x111 (273)
//	gms_v72   0x566768                                    0x135 (309)
//	gms_v79   0x57f3d7                                    0x142 (322)
//	gms_v83   0x5a4241                                    0x15A (346)
//	gms_v84   0x5b46f8  (sub_5B46BC case 0x164)           0x164 (356)
//	gms_v87   0x5d4300                                    0x16F (367)
//	gms_v95   0x575bc0                                    0x19A (410)

// mtsChargeVariant is one implemented (region, major, minor) the writer is routed
// in. The bodiless wire is identical across all of them.
type mtsChargeVariant struct {
	name   string
	region string
	major  uint16
	minor  uint16
}

var mtsChargeVariants = []mtsChargeVariant{
	{"gms_v61", "GMS", 61, 1},
	{"gms_v72", "GMS", 72, 1},
	{"gms_v79", "GMS", 79, 1},
	{"gms_v83", "GMS", 83, 1},
	{"gms_v84", "GMS", 84, 1},
	{"gms_v87", "GMS", 87, 1},
	{"gms_v95", "GMS", 95, 0},
}

// TestMtsChargeParamResultGolden pins the exact wire: an EMPTY body, in every
// version the writer is implemented in. The client read-site is bodiless (IDA
// addresses above), so the golden bytes are the empty slice.
//
// packet-audit:verify packet=field/clientbound/MtsChargeParamResult version=gms_v61 ida=0x52d691
// packet-audit:verify packet=field/clientbound/MtsChargeParamResult version=gms_v72 ida=0x566768
// packet-audit:verify packet=field/clientbound/MtsChargeParamResult version=gms_v79 ida=0x57f3d7
// packet-audit:verify packet=field/clientbound/MtsChargeParamResult version=gms_v83 ida=0x5a4241
// packet-audit:verify packet=field/clientbound/MtsChargeParamResult version=gms_v84 ida=0x5b46f8
// packet-audit:verify packet=field/clientbound/MtsChargeParamResult version=gms_v87 ida=0x5d4300
// packet-audit:verify packet=field/clientbound/MtsChargeParamResult version=gms_v95 ida=0x575bc0
func TestMtsChargeParamResultGolden(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range mtsChargeVariants {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, v.minor)
			b := NewMtsChargeParamResult().Encode(l, ctx)(nil)
			if len(b) != 0 {
				t.Errorf("%s: expected empty (bodiless) charge-param-result body, got %d bytes: %v", v.name, len(b), b)
			}
		})
	}
}

// TestMtsChargeParamResultRoundTrip proves the bodiless packet round-trips: Decode
// reads nothing and leaves an empty buffer, and Encode emits nothing.
func TestMtsChargeParamResultRoundTrip(t *testing.T) {
	input := NewMtsChargeParamResult()
	for _, v := range mtsChargeVariants {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, v.minor)
			output := MtsChargeParamResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
