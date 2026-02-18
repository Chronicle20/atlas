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
	cr := getCharacterRegistry()
	instanceId := uuid.New()

	cr.Add(42, instanceId)

	assert.True(t, cr.IsInTransport(42))
}

func TestCharacterRegistry_Remove(t *testing.T) {
	setupCharacterTestRegistry(t)
	cr := getCharacterRegistry()
	instanceId := uuid.New()

	cr.Add(42, instanceId)
	cr.Remove(42)

	assert.False(t, cr.IsInTransport(42))
}

func TestCharacterRegistry_GetInstanceForCharacter(t *testing.T) {
	setupCharacterTestRegistry(t)
	cr := getCharacterRegistry()
	instanceId := uuid.New()

	cr.Add(42, instanceId)

	id, ok := cr.GetInstanceForCharacter(42)
	assert.True(t, ok)
	assert.Equal(t, instanceId, id)
}

func TestCharacterRegistry_GetInstanceForCharacter_NotFound(t *testing.T) {
	setupCharacterTestRegistry(t)
	cr := getCharacterRegistry()

	_, ok := cr.GetInstanceForCharacter(42)
	assert.False(t, ok)
}

func TestCharacterRegistry_IsInTransport_False(t *testing.T) {
	setupCharacterTestRegistry(t)
	cr := getCharacterRegistry()
	assert.False(t, cr.IsInTransport(42))
}

func TestCharacterRegistry_MultipleCharacters(t *testing.T) {
	setupCharacterTestRegistry(t)
	cr := getCharacterRegistry()
	inst1 := uuid.New()
	inst2 := uuid.New()

	cr.Add(1, inst1)
	cr.Add(2, inst2)
	cr.Add(3, inst1)

	id1, ok1 := cr.GetInstanceForCharacter(1)
	assert.True(t, ok1)
	assert.Equal(t, inst1, id1)

	id2, ok2 := cr.GetInstanceForCharacter(2)
	assert.True(t, ok2)
	assert.Equal(t, inst2, id2)

	id3, ok3 := cr.GetInstanceForCharacter(3)
	assert.True(t, ok3)
	assert.Equal(t, inst1, id3)
}
