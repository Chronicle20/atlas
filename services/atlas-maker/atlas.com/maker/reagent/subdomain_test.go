package reagent_test

import (
	"atlas-maker/reagent"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// catalogPayload is the exact shape of a file under
// deploy/seed/<region>/<version>/reagents — here reagent-4251202.json, the
// negative-value row.
const catalogPayload = `{
  "data": {
    "type": "reagent",
    "id": "4251202",
    "attributes": {
      "stat": "incReqLevel",
      "value": -3
    }
  }
}`

func TestSubdomainSeedsACatalogFile(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, reagent.Migration)
	tenantId := uuid.New()
	te, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)

	sd := reagent.Subdomain{}
	require.Regexp(t, sd.EntityIDPattern(), "reagent-4251202.json")

	attrs, err := sd.Decode([]byte(catalogPayload))
	require.NoError(t, err)
	assert.Equal(t, "incReqLevel", attrs.Stat)
	assert.Equal(t, int16(-3), attrs.Value)

	ms, err := sd.Build(te, "4251202", attrs)
	require.NoError(t, err)
	require.Len(t, ms, 1)
	require.NoError(t, sd.BulkCreate(db, ms))

	m, err := reagent.NewProcessor(testLogger(), databasetest.TenantContext(tenantId), db).GetByItemId(item.Id(4251202))
	require.NoError(t, err)
	assert.Equal(t, "incReqLevel", m.Stat())
	assert.Equal(t, int16(-3), m.Value())

	count, _, err := sd.Count(db)
	require.NoError(t, err)
	assert.EqualValues(t, 1, count)
}

// TestSubdomainRejectsAnUnknownStat keeps a hand-edited catalog file from
// seeding a stat the client has no field for.
func TestSubdomainRejectsAnUnknownStat(t *testing.T) {
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)

	_, err = reagent.Subdomain{}.Build(te, "4251202", reagent.ReagentAttributes{Stat: "incWATK", Value: 1})
	assert.Error(t, err)
}
