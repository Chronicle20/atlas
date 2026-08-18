package monster

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// TestInMapRectUrlUsesMaxNotLimit guards the task-200 live defect: this URL is
// drained through requests.DrainProvider, which appends page[number]/page[size],
// and the server rejects ANY request that also carries a `limit` param
// (paginate.ParseParams -- the task-117 repo-wide ban). Spelling the result cap
// `limit` therefore 400'd every rect query, silently, forever.
//
// Asserted on the built string rather than on a live call so the guard holds
// without a server: the two things that can regress are the param NAME and its
// presence, and both are visible here.
func TestInMapRectUrlUsesMaxNotLimit(t *testing.T) {
	t.Setenv("BASE_SERVICE_URL", "http://monsters/api/")

	f := field.NewBuilder(0, 0, 240011000).SetInstance(uuid.Nil).Build()
	raw, err := inMapRectUrl(context.Background(), f, 488, -628, 888, -328, 7)
	require.NoError(t, err)

	require.NotContains(t, raw, "limit=",
		"a `limit` query param makes atlas-monsters answer 400 for every rect lookup")

	q, err := url.Parse(raw)
	require.NoError(t, err)
	require.Equal(t, "7", q.Query().Get("max"), "the result cap must travel as `max`")

	// The rect bounds must survive verbatim -- a cap rename must not disturb them.
	require.Equal(t, "488", q.Query().Get("x1"))
	require.Equal(t, "-628", q.Query().Get("y1"))
	require.Equal(t, "888", q.Query().Get("x2"))
	require.Equal(t, "-328", q.Query().Get("y2"))
	require.True(t, strings.Contains(raw, "/monsters/in-rect?"))
}
