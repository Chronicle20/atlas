package skillavailability

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// newTestContext builds a tenant-scoped context via the project's Builder
// pattern (tenant.Create + tenant.WithContext -- see job/rest_test.go for
// the sibling usage), so Processor.GetAvailable resolves against the
// requested (region, major, minor) exactly as the live handler does. Kept
// here rather than as a *_testhelpers.go file per CLAUDE.md's test helper
// convention: it is a small in-package function, not a standalone file of
// test-only constructors.
func newTestContext(t *testing.T, region string, major, minor uint16) context.Context {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), region, major, minor)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tn)
}

// TestGetAvailable_V48HasGmHideAtV48WireId pins the pre-Big-Bang wire
// binding for a divergent GM/SuperGM skill: at GMS 48.1, the SuperGmHide
// identity is bound to wire id 5101004 with display name "Super Gm Hide"
// (libs/atlas-constants/skill/version_gms_48_1_gen.go -- SuperGmHide:
// 5101004 in identityToWire_gms_48_1, "Super Gm Hide" in names_gms_48_1,
// present in available_gms_48_1). Exercised via the Processor directly
// (not the paginated HTTP endpoint) because SuperGmHide's high wire id
// sorts well past the default 50-item page.
func TestGetAvailable_V48HasGmHideAtV48WireId(t *testing.T) {
	l, _ := test.NewNullLogger()
	ctx := newTestContext(t, "GMS", 48, 1)

	ms := NewProcessor(l, ctx).GetAvailable()

	found := false
	for _, m := range ms {
		if m.Id == 5101004 {
			found = true
			require.Equal(t, "Super Gm Hide", m.Name)
		}
	}
	require.True(t, found, "expected wire id 5101004 (Super Gm Hide) in v48 availability list")
}

// TestGetAvailable_V72HasGmHideAtV72WireId pins the same identity's
// post-Big-Bang-era wire binding: at GMS 72.1, SuperGmHide is bound to wire
// id 9101004 (libs/atlas-constants/skill/version_gms_72_1_gen.go --
// SuperGmHide: 9101004 in identityToWire_gms_72_1, present in
// available_gms_72_1) -- the same identity, a different wire id, exactly
// the two-axis divergence task-187 exists to make explicit.
func TestGetAvailable_V72HasGmHideAtV72WireId(t *testing.T) {
	l, _ := test.NewNullLogger()
	ctx := newTestContext(t, "GMS", 72, 1)

	ms := NewProcessor(l, ctx).GetAvailable()

	found := false
	for _, m := range ms {
		if m.Id == 9101004 {
			found = true
			require.Equal(t, "Super Gm Hide", m.Name)
		}
	}
	require.True(t, found, "expected wire id 9101004 (Super Gm Hide) in v72 availability list")
}

// TestGetAvailable_VersionStableSkillAppearsInBoth pins a version-stable
// identity as a control: BeginnerThreeSnails binds to the same wire id 1000
// with the same name in both v48 and v72 (version_gms_48_1_gen.go /
// version_gms_72_1_gen.go), unlike SuperGmHide above.
func TestGetAvailable_VersionStableSkillAppearsInBoth(t *testing.T) {
	l, _ := test.NewNullLogger()

	for _, v := range []struct {
		major, minor uint16
	}{{48, 1}, {72, 1}} {
		ctx := newTestContext(t, "GMS", v.major, v.minor)
		ms := NewProcessor(l, ctx).GetAvailable()

		found := false
		for _, m := range ms {
			if m.Id == 1000 {
				found = true
				require.Equal(t, "Beginner Three Snails", m.Name)
			}
		}
		require.True(t, found, "expected wire id 1000 (Beginner Three Snails) at v%d.%d", v.major, v.minor)
	}
}
