package main

import (
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/gen/wzsnapshot"

	"github.com/stretchr/testify/require"
)

// A snapshot written by mksnapshot must load cleanly through
// wzsnapshot.LoadSnapshot, which recomputes and verifies the hash. Sorting
// and de-duplication happen here so the persisted arrays match what
// LoadSnapshot hashes.
func TestCanonicalize_SortsDedupsAndPinsHash(t *testing.T) {
	raw := `{"region":"gms","major":95,"minor":1,"skills":[1002,1000,1002,8],"jobs":[200,100,100]}`

	out, err := canonicalize(strings.NewReader(raw))
	require.NoError(t, err)

	require.Equal(t, "gms", out.Region)
	require.Equal(t, uint16(95), out.Major)
	require.Equal(t, uint16(1), out.Minor)
	require.Equal(t, []uint32{8, 1000, 1002}, out.Skills)
	require.Equal(t, []uint16{100, 200}, out.Jobs)
	require.Equal(t, wzsnapshot.HashIds(out.Skills, out.Jobs), out.Hash)
}

func TestCanonicalize_RejectsEmptySkillSet(t *testing.T) {
	raw := `{"region":"gms","major":95,"minor":1,"skills":[],"jobs":[100]}`

	_, err := canonicalize(strings.NewReader(raw))
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty skill set")
}

func TestRender_IsStableAndNewlineTerminated(t *testing.T) {
	raw := `{"region":"jms","major":185,"minor":1,"skills":[5],"jobs":[1]}`

	out, err := canonicalize(strings.NewReader(raw))
	require.NoError(t, err)

	b, err := render(out)
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(string(b), "\n"))
	require.Contains(t, string(b), `"region": "jms"`)

	again, err := render(out)
	require.NoError(t, err)
	require.Equal(t, string(b), string(again))
}
