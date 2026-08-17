package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func setupTestRedis(t *testing.T) (*goredis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return client, mr
}

func setupTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	data := fmt.Sprintf(`{"id":"%s","region":"GMS","majorVersion":83,"minorVersion":1}`, uuid.New().String())
	var ten tenant.Model
	require.NoError(t, json.Unmarshal([]byte(data), &ten))
	return ten
}

func setupTestContext(t *testing.T, ten tenant.Model) context.Context {
	t.Helper()
	return tenant.WithContext(context.Background(), ten)
}

func TestInitiate_ThenGetExpired_ReturnsAgreement(t *testing.T) {
	client, _ := setupTestRedis(t)
	InitRegistry(client)

	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	ch := channel.NewModel(0, 0)
	leaderId := uint32(100)
	members := []uint32{100, 200, 300}

	err := GetRegistry().Initiate(ctx, ch, "TestGuild", leaderId, members)
	require.NoError(t, err)

	// timeout=0 means any agreement whose age > 0 is expired; since the
	// agreement was created at time.Now(), now.Sub(age) is ~0 which is NOT
	// strictly > 0.  Use a tiny sleep so the age is at least 1 nanosecond old.
	time.Sleep(time.Millisecond)

	expired, err := GetRegistry().GetExpiredAcrossTenants(0)
	require.NoError(t, err)
	require.Len(t, expired, 1)
	assert.Equal(t, "TestGuild", expired[0].Name())
	assert.Equal(t, leaderId, expired[0].LeaderId())
}

func TestRespond_Disagree_RemovesAgreement(t *testing.T) {
	client, _ := setupTestRedis(t)
	InitRegistry(client)

	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	ch := channel.NewModel(0, 0)
	leaderId := uint32(100)
	members := []uint32{100, 200, 300}

	err := GetRegistry().Initiate(ctx, ch, "TestGuild", leaderId, members)
	require.NoError(t, err)

	// Disagreeing should remove the agreement.
	mdl, err := GetRegistry().Respond(ctx, 200, false)
	require.NoError(t, err)
	assert.Equal(t, "TestGuild", mdl.Name())

	// GetExpired with zero timeout should return nothing now.
	expired, err := GetRegistry().GetExpiredAcrossTenants(0)
	require.NoError(t, err)
	assert.Empty(t, expired)
}

func TestRespond_Agree_UpdatesAgreement(t *testing.T) {
	client, _ := setupTestRedis(t)
	InitRegistry(client)

	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	ch := channel.NewModel(0, 0)
	leaderId := uint32(100)
	members := []uint32{100, 200}

	err := GetRegistry().Initiate(ctx, ch, "TestGuild", leaderId, members)
	require.NoError(t, err)

	mdl, err := GetRegistry().Respond(ctx, 200, true)
	require.NoError(t, err)
	assert.True(t, mdl.Responses()[200])
}

func TestGetExpiredAcrossTenants_TwoTenants_Isolation(t *testing.T) {
	client, _ := setupTestRedis(t)
	InitRegistry(client)

	t1 := setupTestTenant(t)
	t2 := setupTestTenant(t)
	ctx1 := setupTestContext(t, t1)
	ctx2 := setupTestContext(t, t2)

	ch := channel.NewModel(0, 0)

	require.NoError(t, GetRegistry().Initiate(ctx1, ch, "Guild1", 100, []uint32{100, 200}))
	require.NoError(t, GetRegistry().Initiate(ctx2, ch, "Guild2", 300, []uint32{300, 400}))

	// Let both agreements age past a zero timeout.
	time.Sleep(time.Millisecond)

	// t1's agreements are stored under t1's tenant key; the agreement created
	// under t2 must not appear when read back through the agreements
	// registry for t1's tenant scope.
	_, err := GetRegistry().agreements.Get(ctx1, t1, mustAgreementId(t, GetRegistry(), ctx1, 100))
	require.NoError(t, err)

	g2ID := mustAgreementId(t, GetRegistry(), ctx2, 300)
	_, err = GetRegistry().agreements.Get(ctx1, t1, g2ID)
	require.Error(t, err, "t1 must not be able to read t2's agreement via t1's tenant scope")

	// GetExpiredAcrossTenants sweeps every tenant and must see both.
	expired, err := GetRegistry().GetExpiredAcrossTenants(0)
	require.NoError(t, err)
	require.Len(t, expired, 2)
	names := map[string]bool{}
	for _, g := range expired {
		names[g.Name()] = true
	}
	assert.True(t, names["Guild1"])
	assert.True(t, names["Guild2"])
}

// mustAgreementId recovers the agreement UUID tracked for characterId under
// ctx, by way of the char->agreement-id index that Initiate populates.
func mustAgreementId(t *testing.T, r *Registry, ctx context.Context, characterId uint32) uuid.UUID {
	t.Helper()
	ten := tenant.MustFromContext(ctx)
	idStr, err := r.charAgree.Get(ctx, ten, characterId)
	require.NoError(t, err)
	id, err := uuid.Parse(idStr)
	require.NoError(t, err)
	return id
}

func TestInitiate_AlreadyInAgreement_ReturnsError(t *testing.T) {
	client, _ := setupTestRedis(t)
	InitRegistry(client)

	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)

	ch := channel.NewModel(0, 0)
	members := []uint32{100, 200}

	// First initiation.
	err := GetRegistry().Initiate(ctx, ch, "Guild1", 100, members)
	require.NoError(t, err)

	// Second initiation with overlapping member should fail.
	err = GetRegistry().Initiate(ctx, ch, "Guild2", 100, members)
	assert.Error(t, err)
}
