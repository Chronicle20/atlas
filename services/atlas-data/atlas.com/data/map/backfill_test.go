package _map

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type backfillServer struct{}

func (backfillServer) GetBaseURL() string { return "" }
func (backfillServer) GetPrefix() string  { return "/api/" }

func seedDoc(t *testing.T, db *gorm.DB, ctx context.Context, tn tenant.Model, id uint32, name, street string) {
	t.Helper()
	rm := RestModel{Name: name, StreetName: street}
	require.NoError(t, rm.SetID(strconv.Itoa(int(id))))
	d, err := jsonapi.MarshalToStruct(rm, backfillServer{})
	require.NoError(t, err)
	raw, err := json.Marshal(d)
	require.NoError(t, err)

	doc := testDocumentEntity{
		Id:         uuid.New(),
		TenantId:   tn.Id(),
		Type:       "MAP",
		DocumentId: id,
		Content:    raw,
	}
	require.NoError(t, db.WithContext(ctx).Create(&doc).Error)
}

func TestBackfill_Populates50Rows(t *testing.T) {
	db := setupStorageTestDB(t)
	l, _ := test.NewNullLogger()

	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	for i := 0; i < 50; i++ {
		seedDoc(t, db, ctx, tn, uint32(1000+i), namef(i), streetf(i))
	}

	res, err := Backfill(l)(ctx)(db)
	require.NoError(t, err)
	assert.Equal(t, 50, res.Processed)

	var idxCount int64
	require.NoError(t, db.WithContext(ctx).Model(&testSearchIndexEntity{}).Count(&idxCount).Error)
	assert.Equal(t, int64(50), idxCount)

	var sample testSearchIndexEntity
	require.NoError(t, db.WithContext(ctx).Where("map_id = ?", 1000).First(&sample).Error)
	assert.Equal(t, namef(0), sample.Name)
	assert.Equal(t, streetf(0), sample.StreetName)
}

func TestBackfill_Idempotent(t *testing.T) {
	db := setupStorageTestDB(t)
	l, _ := test.NewNullLogger()

	tn := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)

	for i := 0; i < 50; i++ {
		seedDoc(t, db, ctx, tn, uint32(2000+i), namef(i), streetf(i))
	}

	_, err := Backfill(l)(ctx)(db)
	require.NoError(t, err)

	var first testSearchIndexEntity
	require.NoError(t, db.WithContext(ctx).Where("map_id = ?", 2000).First(&first).Error)
	firstTime := first.UpdatedAt

	time.Sleep(10 * time.Millisecond)

	res2, err := Backfill(l)(ctx)(db)
	require.NoError(t, err)
	assert.Equal(t, 50, res2.Processed)

	var count int64
	require.NoError(t, db.WithContext(ctx).Model(&testSearchIndexEntity{}).Count(&count).Error)
	assert.Equal(t, int64(50), count, "row count must stay stable across runs")

	var second testSearchIndexEntity
	require.NoError(t, db.WithContext(ctx).Where("map_id = ?", 2000).First(&second).Error)
	assert.True(t, !second.UpdatedAt.Before(firstTime), "updated_at must advance (or stay equal) across runs")
}

func namef(i int) string {
	return "Map " + padN(i)
}

func streetf(i int) string {
	return "Street " + padN(i)
}

func padN(i int) string {
	s := ""
	n := i
	if n == 0 {
		return "0"
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
