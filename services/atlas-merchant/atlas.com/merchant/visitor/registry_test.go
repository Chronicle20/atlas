package visitor

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*goredis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return client, mr
}

func makeTenant(id string, region string, major uint16, minor uint16) tenant.Model {
	data := fmt.Sprintf(`{"id":"%s","region":"%s","majorVersion":%d,"minorVersion":%d}`, id, region, major, minor)
	var t tenant.Model
	_ = json.Unmarshal([]byte(data), &t)
	return t
}

func setupTestRegistry(t *testing.T) (*Registry, tenant.Model) {
	t.Helper()
	client, _ := setupTestRedis(t)
	InitRegistry(client)
	ten := makeTenant("00000000-0000-0000-0000-000000000001", "GMS", 83, 1)
	return GetRegistry(), ten
}

func TestAddVisitor_InsertionOrder(t *testing.T) {
	r, ten := setupTestRegistry(t)
	ctx := context.Background()
	shopId := uuid.New()

	// Add two characters; scores are insertion timestamps so they come back in add order.
	require.NoError(t, r.AddVisitor(ctx, ten, shopId, 1001))
	require.NoError(t, r.AddVisitor(ctx, ten, shopId, 1002))

	visitors, err := r.GetVisitors(ctx, ten, shopId)
	require.NoError(t, err)
	assert.Equal(t, []uint32{1001, 1002}, visitors)

	count, err := r.GetVisitorCount(ctx, ten, shopId)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestRemoveVisitor(t *testing.T) {
	r, ten := setupTestRegistry(t)
	ctx := context.Background()
	shopId := uuid.New()

	require.NoError(t, r.AddVisitor(ctx, ten, shopId, 1001))
	require.NoError(t, r.AddVisitor(ctx, ten, shopId, 1002))

	require.NoError(t, r.RemoveVisitor(ctx, ten, shopId, 1001))

	count, err := r.GetVisitorCount(ctx, ten, shopId)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	visitors, err := r.GetVisitors(ctx, ten, shopId)
	require.NoError(t, err)
	assert.Equal(t, []uint32{1002}, visitors)
}

func TestGetShopForCharacter(t *testing.T) {
	r, ten := setupTestRegistry(t)
	ctx := context.Background()
	shopId := uuid.New()

	require.NoError(t, r.AddVisitor(ctx, ten, shopId, 1001))

	resolved, err := r.GetShopForCharacter(ctx, ten, 1001)
	require.NoError(t, err)
	assert.Equal(t, shopId, resolved)
}

func TestRemoveAllVisitors(t *testing.T) {
	r, ten := setupTestRegistry(t)
	ctx := context.Background()
	shopId := uuid.New()

	require.NoError(t, r.AddVisitor(ctx, ten, shopId, 1001))
	require.NoError(t, r.AddVisitor(ctx, ten, shopId, 1002))

	evicted, err := r.RemoveAllVisitors(ctx, ten, shopId)
	require.NoError(t, err)
	assert.ElementsMatch(t, []uint32{1001, 1002}, evicted)

	count, err := r.GetVisitorCount(ctx, ten, shopId)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestShopsAreIndependent(t *testing.T) {
	r, ten := setupTestRegistry(t)
	ctx := context.Background()
	shopA := uuid.New()
	shopB := uuid.New()

	require.NoError(t, r.AddVisitor(ctx, ten, shopA, 1001))
	require.NoError(t, r.AddVisitor(ctx, ten, shopB, 2001))

	countA, err := r.GetVisitorCount(ctx, ten, shopA)
	require.NoError(t, err)
	assert.Equal(t, 1, countA)

	countB, err := r.GetVisitorCount(ctx, ten, shopB)
	require.NoError(t, err)
	assert.Equal(t, 1, countB)

	// Clearing shopA does not affect shopB.
	_, err = r.RemoveAllVisitors(ctx, ten, shopA)
	require.NoError(t, err)

	countA, err = r.GetVisitorCount(ctx, ten, shopA)
	require.NoError(t, err)
	assert.Equal(t, 0, countA)

	countB, err = r.GetVisitorCount(ctx, ten, shopB)
	require.NoError(t, err)
	assert.Equal(t, 1, countB)
}

func TestNewKeyFormat_EnvPrefixed(t *testing.T) {
	client, mr := setupTestRedis(t)
	InitRegistry(client)
	ten := makeTenant("00000000-0000-0000-0000-000000000002", "GMS", 83, 1)
	ctx := context.Background()
	shopId := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	require.NoError(t, GetRegistry().AddVisitor(ctx, ten, shopId, 9999))

	tenKey := atlas.TenantKey(ten)
	wantKey := atlas.KeyPrefix() + ":merchant:shop-visitors:" + tenKey + ":" + shopId.String()
	assert.True(t, mr.Exists(wantKey), "expected key %q to exist; keys=%v", wantKey, mr.Keys())
}
