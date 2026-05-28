package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
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

	expired, err := GetRegistry().GetExpired(0)
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
	expired, err := GetRegistry().GetExpired(0)
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
