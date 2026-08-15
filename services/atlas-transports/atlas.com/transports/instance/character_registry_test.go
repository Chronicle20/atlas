package instance

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupCharacterTestRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitCharacterRegistry(rc)
}

func TestCharacterRegistry_Add(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := newTestTenantContext(t)
	cr := getCharacterRegistry()
	instanceId := uuid.New()

	cr.Add(ctx, 42, instanceId)

	assert.True(t, cr.IsInTransport(ctx, 42))
}

func TestCharacterRegistry_Remove(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := newTestTenantContext(t)
	cr := getCharacterRegistry()
	instanceId := uuid.New()

	cr.Add(ctx, 42, instanceId)
	cr.Remove(ctx, 42)

	assert.False(t, cr.IsInTransport(ctx, 42))
}

func TestCharacterRegistry_GetInstanceForCharacter(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := newTestTenantContext(t)
	cr := getCharacterRegistry()
	instanceId := uuid.New()

	cr.Add(ctx, 42, instanceId)

	id, ok := cr.GetInstanceForCharacter(ctx, 42)
	assert.True(t, ok)
	assert.Equal(t, instanceId, id)
}

func TestCharacterRegistry_GetInstanceForCharacter_NotFound(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := newTestTenantContext(t)
	cr := getCharacterRegistry()

	_, ok := cr.GetInstanceForCharacter(ctx, 42)
	assert.False(t, ok)
}

func TestCharacterRegistry_IsInTransport_False(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := newTestTenantContext(t)
	cr := getCharacterRegistry()
	assert.False(t, cr.IsInTransport(ctx, 42))
}

func TestCharacterRegistry_MultipleCharacters(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctx := newTestTenantContext(t)
	cr := getCharacterRegistry()
	inst1 := uuid.New()
	inst2 := uuid.New()

	cr.Add(ctx, 1, inst1)
	cr.Add(ctx, 2, inst2)
	cr.Add(ctx, 3, inst1)

	id1, ok1 := cr.GetInstanceForCharacter(ctx, 1)
	assert.True(t, ok1)
	assert.Equal(t, inst1, id1)

	id2, ok2 := cr.GetInstanceForCharacter(ctx, 2)
	assert.True(t, ok2)
	assert.Equal(t, inst2, id2)

	id3, ok3 := cr.GetInstanceForCharacter(ctx, 3)
	assert.True(t, ok3)
	assert.Equal(t, inst1, id3)
}

// TestCharacterRegistry_TwoTenantsSameCharacterIdDoNotCollide is the live
// production bug this task fixes: character ids are per-tenant sequences, so
// tenant A's character 12345 and tenant B's character 12345 must resolve to
// independent instances.
func TestCharacterRegistry_TwoTenantsSameCharacterIdDoNotCollide(t *testing.T) {
	setupCharacterTestRegistry(t)
	ctxA := newTestTenantContext(t)
	ctxB := newTestTenantContext(t)
	cr := getCharacterRegistry()

	instA := uuid.New()
	instB := uuid.New()

	cr.Add(ctxA, 12345, instA)
	cr.Add(ctxB, 12345, instB)

	idA, okA := cr.GetInstanceForCharacter(ctxA, 12345)
	assert.True(t, okA)
	assert.Equal(t, instA, idA)

	idB, okB := cr.GetInstanceForCharacter(ctxB, 12345)
	assert.True(t, okB)
	assert.Equal(t, instB, idB)

	// Removing tenant A's character must not affect tenant B's.
	cr.Remove(ctxA, 12345)
	assert.False(t, cr.IsInTransport(ctxA, 12345))
	assert.True(t, cr.IsInTransport(ctxB, 12345))
}
