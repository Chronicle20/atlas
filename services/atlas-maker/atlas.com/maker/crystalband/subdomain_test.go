package crystalband_test

import (
	"atlas-maker/crystalband"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// catalogPayload is the exact shape of a file under
// deploy/seed/<region>/<version>/crystal-bands — here crystal-band-31.json,
// the lowest of the nine derived bands.
const catalogPayload = `{
  "data": {
    "type": "crystalBand",
    "id": "31",
    "attributes": {
      "maxLevel": 50,
      "crystalItemId": 4260000,
      "count": 1
    }
  }
}`

func TestSubdomainSeedsACatalogFile(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, crystalband.Migration)
	tenantId := uuid.New()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)

	sd := crystalband.Subdomain{}
	require.Regexp(t, sd.EntityIDPattern(), "crystal-band-31.json")

	attrs, err := sd.Decode([]byte(catalogPayload))
	require.NoError(t, err)
	assert.EqualValues(t, 50, attrs.MaxLevel)
	assert.EqualValues(t, 4260000, attrs.CrystalItemId)
	assert.EqualValues(t, 1, attrs.Count)

	ms, err := sd.Build(te, "31", attrs)
	require.NoError(t, err)
	require.Len(t, ms, 1)
	require.NoError(t, sd.BulkCreate(db, ms))

	m, err := crystalband.NewProcessor(testLogger(), databasetest.TenantContext(tenantId), db).GetByMinLevel(31)
	require.NoError(t, err)
	assert.EqualValues(t, 50, m.MaxLevel())

	count, _, err := sd.Count(db)
	require.NoError(t, err)
	assert.EqualValues(t, 1, count)
}

// TestSubdomainRejectsAnInvertedRange keeps a hand-edited catalog file from
// seeding a band whose max is below its min.
func TestSubdomainRejectsAnInvertedRange(t *testing.T) {
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)

	_, err = crystalband.Subdomain{}.Build(te, "50", crystalband.CrystalBandAttributes{MaxLevel: 31, CrystalItemId: 4260000, Count: 1})
	assert.Error(t, err)
}
