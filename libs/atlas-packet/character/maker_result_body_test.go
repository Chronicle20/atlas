package character

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// makerResultOptions builds a tenant "operations" table in the JSON-decoded
// shape ResolveCode consumes (float64 values), mapping the arm key to a
// deliberately NON-default mode byte.
func makerResultOptions(key string, mode float64) map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{key: mode},
	}
}

func encodeBody(t *testing.T, ctx context.Context, body func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte, options map[string]interface{}) []byte {
	t.Helper()
	l, _ := testlog.NewNullLogger()
	return body(l, ctx)(options)
}

// TestMakerResultBodyFunctionsResolveMode is the executable proof that every
// mode-carrying MAKER_RESULT arm takes its nMode from the tenant "operations"
// table at emit time (DOM-25) rather than from a Go literal. Each arm is emitted
// with its key mapped to 7 — a value the client has no arm for — and the i32 at
// offset 4 (immediately after nResult) must be 7. A hard-coded mode would emit
// its own literal here and fail.
func TestMakerResultBodyFunctionsResolveMode(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	materials := []clientbound.MakerMaterial{clientbound.NewMakerMaterial(4011001, 5)}

	cases := []struct {
		name string
		key  string
		body func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte
	}{
		{
			"Create", MakerResultOperationCreate,
			MakerResultCreateBody(0, false, 1082002, 1, materials, []uint32{4021313}, true, 4130000, 1200),
		},
		{
			"CreateWithUpgrade", MakerResultOperationCreateWithUpgrade,
			MakerResultCreateWithUpgradeBody(0, false, 1082002, 1, materials, []uint32{4021313}, true, 4130000, 1200),
		},
		{
			"MonsterCrystal", MakerResultOperationMonsterCrystal,
			MakerResultMonsterCrystalBody(0, 4000000, 4000001),
		},
		{
			"Disassemble", MakerResultOperationDisassemble,
			MakerResultDisassembleBody(0, 1082002, materials, 500),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := encodeBody(t, ctx, c.body, makerResultOptions(c.key, 7))
			if len(out) < 8 {
				t.Fatalf("encoded %d bytes, want at least 8", len(out))
			}
			// nResult occupies [0:4]; nMode is the i32 at [4:8].
			if out[4] != 0x07 || out[5] != 0x00 || out[6] != 0x00 || out[7] != 0x00 {
				t.Errorf("nMode = % X, want 07 00 00 00 (mode must be config-resolved, not a literal)", out[4:8])
			}
			if out[0] != 0x00 || out[1] != 0x00 || out[2] != 0x00 || out[3] != 0x00 {
				t.Errorf("nResult = % X, want 00 00 00 00", out[0:4])
			}
		})
	}
}

// TestMakerResultFailedBodyWritesOnlyResult pins the bodyless arm: it resolves
// nothing from the operations table (there is no FAILED key — see
// docs/packets/dispatchers/maker_result.yaml) and writes nResult alone. An
// options table carrying a stray FAILED value must not change the four bytes.
func TestMakerResultFailedBodyWritesOnlyResult(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	out := encodeBody(t, ctx, MakerResultFailedBody(2), makerResultOptions(MakerResultOperationFailed, 7))
	if len(out) != 4 {
		t.Fatalf("encoded %d bytes, want 4 (nResult only, no mode field)", len(out))
	}
	if out[0] != 0x02 || out[1] != 0x00 || out[2] != 0x00 || out[3] != 0x00 {
		t.Errorf("nResult = % X, want 02 00 00 00", out)
	}
	// Also with no operations table at all: the arm resolves nothing, so it
	// must not fall back to ResolveCode's 99 sentinel.
	out = encodeBody(t, ctx, MakerResultFailedBody(2), nil)
	if len(out) != 4 || out[0] != 0x02 {
		t.Errorf("bodyless arm with nil options = % X, want 02 00 00 00", out)
	}
}
