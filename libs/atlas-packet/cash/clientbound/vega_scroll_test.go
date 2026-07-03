package clientbound

import (
	"encoding/hex"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// Per-version VegaScroll operations tables, IDA-verified against
// CUIVega::OnVegaResult / CUIVega::Draw per version (task-130 Task 4). On EVERY
// version the success/fail popup (SuccessWnd/FailWnd, EffectSuccess/EffectFail)
// is chosen by the START byte in CUIVega::Draw — the RESULT byte is only
// range-validated — so the START byte carries the outcome on all versions. The
// values are version-shifted (v87=v83+2, v95=v83+4, jms is its own map).
// Template JSON numbers decode as float64.

// vegaOpsV83 — IDA v83 CUIVega::OnVegaResult 0x82d8d5 (start∈{0x40,0x45},
// result∈{0x41,0x43}); CUIVega::Draw sub_82C28B + popup sub_82DA77
// (start 0x40→SP_5398 SUCCESSWND, start 0x45→SP_5399 FAILWND).
func vegaOpsV83() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			"START_SUCCESS":  float64(0x40),
			"START_FAILURE":  float64(0x45),
			"RESULT_SUCCESS": float64(0x41),
			"RESULT_FAILURE": float64(0x43),
			"INVALID":        float64(0x42),
		},
	}
}

// vegaOpsV87 — IDA v87 CUIVega::OnVegaResult 0x8919b6 (start∈{0x42,0x47},
// result∈{0x43,0x45}); Draw sub_890325 + popup sub_891B4E (type1=success).
func vegaOpsV87() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			"START_SUCCESS":  float64(0x42),
			"START_FAILURE":  float64(0x47),
			"RESULT_SUCCESS": float64(0x43),
			"RESULT_FAILURE": float64(0x45),
			"INVALID":        float64(0x44),
		},
	}
}

// vegaOpsV95 — IDA v95 CUIVega::OnVegaResult 0x7bf7b0 (start∈{0x44,0x49},
// result∈{0x45,0x47}); Draw 0x7c1dd0 + popup OnCreate 0x7c2bd0 →
// Effect_Vega 0x457600 (start 0x44→popup type1→EffectSuccess, start
// 0x49→popup type2→EffectFail). START pairing PINNED: 0x44 success / 0x49
// failure (Task 12 copies these verbatim).
func vegaOpsV95() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			"START_SUCCESS":  float64(0x44),
			"START_FAILURE":  float64(0x49),
			"RESULT_SUCCESS": float64(0x45),
			"RESULT_FAILURE": float64(0x47),
			"INVALID":        float64(0x42),
		},
	}
}

// vegaOpsJMS — IDA jms_v185 CUIVega::OnVegaResult 0x8b89ad (start∈{0x3B,0x40},
// result∈{0x3C,0x3E}); Draw sub_8B7378 + popup sub_8B8B45 (type1=success).
func vegaOpsJMS() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			"START_SUCCESS":  float64(0x3B),
			"START_FAILURE":  float64(0x40),
			"RESULT_SUCCESS": float64(0x3C),
			"RESULT_FAILURE": float64(0x3E),
			"INVALID":        float64(0x3D),
		},
	}
}

type vegaVariant struct {
	name          string
	region        string
	major         uint16
	minor         uint16
	ops           func() map[string]interface{}
	startSuccess  byte
	startFailure  byte
	resultSuccess byte
	resultFailure byte
	invalid       byte
}

func vegaVariants() []vegaVariant {
	return []vegaVariant{
		{"gms_v83", "GMS", 83, 1, vegaOpsV83, 0x40, 0x45, 0x41, 0x43, 0x42},
		{"gms_v87", "GMS", 87, 1, vegaOpsV87, 0x42, 0x47, 0x43, 0x45, 0x44},
		{"gms_v95", "GMS", 95, 1, vegaOpsV95, 0x44, 0x49, 0x45, 0x47, 0x42},
		{"jms_v185", "JMS", 185, 1, vegaOpsJMS, 0x3B, 0x40, 0x3C, 0x3E, 0x3D},
	}
}

// TestVegaScrollByteOutput locks the exact wire byte resolved for every
// outcome-keyed operations key on every IDA-verified version. The single mode
// byte is the whole packet body (opcode is the tenant writer opcode; the body
// is one Decode1 on the client — CUIVega::OnVegaResult).
//
// packet-audit:verify packet=cash/clientbound/CashVegaScroll version=gms_v83 ida=0x82d8d5
// packet-audit:verify packet=cash/clientbound/CashVegaScroll version=gms_v87 ida=0x8919b6
// packet-audit:verify packet=cash/clientbound/CashVegaScroll version=gms_v95 ida=0x7bf7b0
// packet-audit:verify packet=cash/clientbound/CashVegaScroll version=jms_v185 ida=0x8b89ad
func TestVegaScrollByteOutput(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	for _, v := range vegaVariants() {
		t.Run(v.name, func(t *testing.T) {
			ctx := pt.CreateContext(v.region, v.major, v.minor)
			ops := v.ops()
			cases := []struct {
				name string
				got  []byte
				want byte
			}{
				{"start success", VegaScrollStartBody(true)(l, ctx)(ops), v.startSuccess},
				{"start failure", VegaScrollStartBody(false)(l, ctx)(ops), v.startFailure},
				{"result success", VegaScrollResultBody(true)(l, ctx)(ops), v.resultSuccess},
				{"result failure", VegaScrollResultBody(false)(l, ctx)(ops), v.resultFailure},
				{"invalid", VegaScrollInvalidBody()(l, ctx)(ops), v.invalid},
			}
			for _, tc := range cases {
				if len(tc.got) != 1 || tc.got[0] != tc.want {
					t.Errorf("%s: got %v, want [%#x]", tc.name, tc.got, tc.want)
				}
			}
			// Audit representative: VegaScroll{mode}.Encode is the single mode
			// byte the client reads via one Decode1 (matches CashVegaScroll report).
			raw := hex.EncodeToString(NewVegaScroll(v.startSuccess).Encode(l, ctx)(nil))
			if raw != hex.EncodeToString([]byte{v.startSuccess}) {
				t.Errorf("audit representative byte: got %s", raw)
			}
		})
	}
}

// A missing operations table must fall back to 99 (ResolveCode contract) —
// this is the misconfigured-tenant canary, not a supported path. 99 (0x63) is
// outside every version's accepted {START,RESULT} set, so the client routes it
// to the safe notice arm on all versions (verified: v83 {0x40,0x41,0x43,0x45},
// v87 {0x42,0x43,0x45,0x47}, v95 {0x44,0x45,0x47,0x49}, jms {0x3B,0x3C,0x3E,0x40}).
func TestVegaScrollBodyMissingOperations(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	got := VegaScrollInvalidBody()(l, ctx)(map[string]interface{}{})
	if len(got) != 1 || got[0] != 99 {
		t.Errorf("missing-operations fallback: got %v, want [99]", got)
	}
}

func TestVegaScrollRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewVegaScrollStart(0x40)
			output := VegaScrollStart{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}
