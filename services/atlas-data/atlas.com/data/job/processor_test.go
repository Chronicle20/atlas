package job

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testCtx(t *testing.T, tenantId uuid.UUID, region string, major, minor uint16) context.Context {
	t.Helper()
	tn, err := tenant.Create(tenantId, region, major, minor)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tn)
}

func TestGetSkillsForJob_ReadsSeededDocument(t *testing.T) {
	db := setupResourceTestDB(t)
	l, _ := test.NewNullLogger()
	tenantId := uuid.New()
	ctx := testCtx(t, tenantId, "GMS", 83, 1)

	_, err := NewStorage(l, db).Add(ctx)(RestModel{Id: 112, Skills: []uint32{1121000, 1121001}})()
	require.NoError(t, err)

	got, ok := NewProcessor(l, ctx, db).GetSkillsForJob(112)
	require.True(t, ok)
	require.Equal(t, uint32(112), got.Id)
	require.Equal(t, []uint32{1121000, 1121001}, got.Skills)
}

func TestGetSkillsForJob_AbsentJobIsNotOk(t *testing.T) {
	db := setupResourceTestDB(t)
	l, _ := test.NewNullLogger()
	ctx := testCtx(t, uuid.New(), "GMS", 83, 1)

	got, ok := NewProcessor(l, ctx, db).GetSkillsForJob(99999)
	require.False(t, ok)
	require.Equal(t, uint32(99999), got.Id)
	require.Empty(t, got.Skills)
}

// Job id 0 (Beginner) is a legitimate document_id — DbStorage.Add derives it by
// strconv.Atoi on GetID() (document/db_storage.go:133), so 0 must round-trip.
func TestGetSkillsForJob_JobIdZeroRoundTrips(t *testing.T) {
	db := setupResourceTestDB(t)
	l, _ := test.NewNullLogger()
	ctx := testCtx(t, uuid.New(), "GMS", 83, 1)

	_, err := NewStorage(l, db).Add(ctx)(RestModel{Id: 0, Skills: []uint32{1000, 1001}})()
	require.NoError(t, err)

	got, ok := NewProcessor(l, ctx, db).GetSkillsForJob(0)
	require.True(t, ok)
	require.Equal(t, uint32(0), got.Id)
	require.Equal(t, []uint32{1000, 1001}, got.Skills)
}

// The headline PRD acceptance criterion: two tenants on different versions get
// different skill lists for the same job id.
func TestGetSkillsForJob_DivergesByTenantVersion(t *testing.T) {
	db := setupResourceTestDB(t)
	l, _ := test.NewNullLogger()

	oldCtx := testCtx(t, uuid.New(), "GMS", 61, 1)
	newCtx := testCtx(t, uuid.New(), "GMS", 95, 1)

	_, err := NewStorage(l, db).Add(oldCtx)(RestModel{Id: 510, Skills: []uint32{5101000, 5101001}})()
	require.NoError(t, err)
	_, err = NewStorage(l, db).Add(newCtx)(RestModel{Id: 510, Skills: []uint32{5101000}})()
	require.NoError(t, err)

	oldGot, ok := NewProcessor(l, oldCtx, db).GetSkillsForJob(510)
	require.True(t, ok)
	newGot, ok := NewProcessor(l, newCtx, db).GetSkillsForJob(510)
	require.True(t, ok)

	require.Equal(t, []uint32{5101000, 5101001}, oldGot.Skills)
	require.Equal(t, []uint32{5101000}, newGot.Skills)
	require.NotEqual(t, oldGot.Skills, newGot.Skills)
}

func TestRegisterJob_WritesOneDocumentAndReturnsCount(t *testing.T) {
	db := setupResourceTestDB(t)
	l, _ := test.NewNullLogger()
	ctx := testCtx(t, uuid.New(), "GMS", 83, 1)

	path := writeTempImage(t, "112.img.xml", jobImageXML)
	n, err := NewProcessor(l, ctx, db).RegisterJob(path)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	got, ok := NewProcessor(l, ctx, db).GetSkillsForJob(112)
	require.True(t, ok)
	require.Equal(t, []uint32{1121000, 1121001, 1121002}, got.Skills)
}

func TestRegisterJob_NonNumericImageWritesNothing(t *testing.T) {
	db := setupResourceTestDB(t)
	l, _ := test.NewNullLogger()
	ctx := testCtx(t, uuid.New(), "GMS", 83, 1)

	path := writeTempImage(t, "MobSkill.img.xml", mobSkillImageXML)
	n, err := NewProcessor(l, ctx, db).RegisterJob(path)
	require.NoError(t, err)
	require.Equal(t, 0, n)
}
