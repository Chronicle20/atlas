package shop

import (
	"testing"

	blacklistpkg "atlas-merchant/blacklist"
	visitpkg "atlas-merchant/visit"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A blacklisted visitor is refused entry (ENTER_FAILED, not admitted); a
// successful entry is recorded for the owner's visit-list; add/remove is
// owner-guarded.
func TestBlacklistEnforcementAndVisitRecording(t *testing.T) {
	db := setupTestDB(t)
	ctx, _ := setupTestContext(t)
	require.NoError(t, blacklistpkg.Migration(db))
	require.NoError(t, visitpkg.Migration(db))
	l, _ := test.NewNullLogger()
	setupTestRegistries(t)
	p := NewProcessor(l, ctx, db).(*ProcessorImpl)

	const owner, stranger = uint32(8000), uint32(8001)
	m, err := p.CreateShop(owner, HiredMerchant, "Merch", 0, 0, 910000001, uuid.Nil, 0, 0, 5030000)
	require.NoError(t, err)
	require.NoError(t, db.WithContext(ctx).Model(&Entity{}).Where("id = ?", m.Id()).Update("state", byte(Open)).Error)

	mb := testBuffer()

	assert.ErrorIs(t, p.AddToBlacklist(mb)(m.Id(), stranger, "Griefer", 0), ErrNotOwner)

	require.NoError(t, p.AddToBlacklist(mb)(m.Id(), owner, "Griefer", 0))
	names, err := p.GetBlacklistPaged(m.Id(), model.Page{Number: 1, Size: 250})
	require.NoError(t, err)
	assert.Equal(t, []string{"Griefer"}, names.Items)
	assert.Equal(t, 1, names.Total)

	require.NoError(t, p.EnterShop(mb)(9001, m.Id(), "Griefer"))
	visitors, err := p.GetVisitors(m.Id())
	require.NoError(t, err)
	assert.NotContains(t, visitors, uint32(9001), "banned visitor not admitted")

	require.NoError(t, p.EnterShop(mb)(9002, m.Id(), "Buyer"))
	visits, err := p.GetVisitsPaged(m.Id(), model.Page{Number: 1, Size: 250})
	require.NoError(t, err)
	require.Len(t, visits.Items, 1)
	assert.Equal(t, "Buyer", visits.Items[0].Name())
	assert.Equal(t, uint32(1), visits.Items[0].Count())

	require.NoError(t, p.RemoveFromBlacklist(mb)(m.Id(), owner, "Griefer"))
	names, err = p.GetBlacklistPaged(m.Id(), model.Page{Number: 1, Size: 250})
	require.NoError(t, err)
	assert.Empty(t, names.Items)
	assert.Equal(t, 0, names.Total)
}
