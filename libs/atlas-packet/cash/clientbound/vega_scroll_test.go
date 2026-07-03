package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// vegaOpsV83 mirrors the gms_83 tenant template operations table (task-130
// design §2.3, IDA-verified CUIVega::OnVegaResult 0x82d8d5). Template JSON
// numbers decode as float64.
func vegaOpsV83() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			"START_SUCCESS":  float64(64),
			"START_FAILURE":  float64(64),
			"RESULT_SUCCESS": float64(65),
			"RESULT_FAILURE": float64(67),
			"INVALID":        float64(66),
		},
	}
}

func TestVegaScrollBodyResolution(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	cases := []struct {
		name string
		body func() []byte
		want byte
	}{
		{"start success", func() []byte { return VegaScrollStartBody(true)(l, ctx)(vegaOpsV83()) }, 0x40},
		{"start failure", func() []byte { return VegaScrollStartBody(false)(l, ctx)(vegaOpsV83()) }, 0x40},
		{"result success", func() []byte { return VegaScrollResultBody(true)(l, ctx)(vegaOpsV83()) }, 0x41},
		{"result failure", func() []byte { return VegaScrollResultBody(false)(l, ctx)(vegaOpsV83()) }, 0x43},
		{"invalid", func() []byte { return VegaScrollInvalidBody()(l, ctx)(vegaOpsV83()) }, 0x42},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.body()
			if len(got) != 1 || got[0] != tc.want {
				t.Errorf("body bytes: got %v, want [%#x]", got, tc.want)
			}
		})
	}
}

// A missing operations table must fall back to 99 (ResolveCode contract) —
// this is the misconfigured-tenant canary, not a supported path.
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
