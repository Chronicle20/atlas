package job

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestStoredContentCarriesNoRelationships is the FR-4.4 / D2 regression guard.
// document.DbStorage.Add persists json.Marshal(jsonapi.MarshalToStruct(m, …))
// (document/db_storage.go:123-130), and api2go populates Document.Included for
// ANY model implementing MarshalIncludedRelations — there is no "only when
// asked" gate at that layer. If someone later merges RestModel and
// ListRestModel into one type, relationship/included data starts leaking into
// the stored `content` column and this test fails.
func TestStoredContentCarriesNoRelationships(t *testing.T) {
	db := setupResourceTestDB(t)
	l, _ := test.NewNullLogger()
	tenantId := uuid.New()
	tn, err := tenant.Create(tenantId, "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tn)

	_, err = NewStorage(l, db).Add(ctx)(RestModel{Id: 112, Skills: []uint32{1121000, 1121001}})()
	require.NoError(t, err)

	var raw string
	require.NoError(t, db.Raw(
		`SELECT content FROM documents WHERE tenant_id = ? AND type = 'JOB' AND document_id = 112`,
		tenantId.String(),
	).Scan(&raw).Error)

	require.NotEmpty(t, raw)
	require.Contains(t, raw, `"skills"`)
	require.False(t, strings.Contains(raw, `"relationships"`), "stored content leaked relationships: %s", raw)
	require.False(t, strings.Contains(raw, `"included"`), "stored content leaked included: %s", raw)
}
