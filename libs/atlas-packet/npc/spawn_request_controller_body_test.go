package npc

import (
	"context"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestNpcControllerGrantBodyResolvesFlag proves NpcControllerGrantBody resolves
// the leading flag byte from the tenant "operations" table (options.operations)
// instead of hard-coding it (DOM-25, task-176).
func TestNpcControllerGrantBodyResolvesFlag(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	ops := map[string]interface{}{
		"operations": map[string]interface{}{
			NpcControllerGrant:  float64(1),
			NpcControllerRevoke: float64(0),
		},
	}

	got := pt.Encode(t, ctx, NpcControllerGrantBody(100, 9010000, 150, -300, 0, 500, -50, 250, true), ops)
	if len(got) == 0 || got[0] != 0x01 {
		t.Fatalf("grant leading byte: got % X want first byte 0x01", got)
	}
}

// TestNpcControllerRevokeBodyResolvesFlag proves NpcControllerRevokeBody
// resolves the leading flag byte from the tenant "operations" table.
func TestNpcControllerRevokeBodyResolvesFlag(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	ops := map[string]interface{}{
		"operations": map[string]interface{}{
			NpcControllerGrant:  float64(1),
			NpcControllerRevoke: float64(0),
		},
	}

	got := pt.Encode(t, ctx, NpcControllerRevokeBody(42), ops)
	want := []byte{0x00, 0x2A, 0x00, 0x00, 0x00}
	if len(got) != len(want) {
		t.Fatalf("revoke encode: got % X want % X", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("revoke encode: got % X want % X", got, want)
		}
	}
}

// TestNpcControllerGrantBodyMissingConfigYields99Sentinel proves a tenant
// missing the "operations" block (unmigrated live config) falls back to the
// ResolveCode 99 client-crash sentinel rather than silently defaulting to a
// hard-coded flag value.
func TestNpcControllerGrantBodyMissingConfigYields99Sentinel(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	body := NpcControllerGrantBody(100, 9010000, 150, -300, 0, 500, -50, 250, true)
	got := body(l, context.Background())(map[string]interface{}{})
	if len(got) == 0 || got[0] != 99 {
		t.Fatalf("missing-config leading byte: got % X want first byte 99 (0x63)", got)
	}
}
