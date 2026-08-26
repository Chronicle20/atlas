package reactor

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestRegistry_TouchLatch(t *testing.T) {
	setupTestRegistry(t)
	ten := setupTestTenant()

	// Step 1: first claim wins.
	assert.True(t, GetRegistry().TryLatchTouch(ten, 42, 1000))

	// Step 2: already latched.
	assert.False(t, GetRegistry().TryLatchTouch(ten, 42, 1000))

	// Step 3: different character latches independently.
	assert.True(t, GetRegistry().TryLatchTouch(ten, 42, 2000))

	// Step 4: clearing one character's latch allows it to be re-claimed.
	GetRegistry().ClearTouch(ten, 42, 1000)
	assert.True(t, GetRegistry().TryLatchTouch(ten, 42, 1000))

	// Step 5: wiping all latches releases every character.
	GetRegistry().ClearAllTouches(ten, 42)
	assert.True(t, GetRegistry().TryLatchTouch(ten, 42, 2000))
}

func TestRegistry_TouchLatchIsTenantScoped(t *testing.T) {
	setupTestRegistry(t)
	tenA := setupTestTenant()
	tenB, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	assert.True(t, GetRegistry().TryLatchTouch(tenA, 42, 1000))
	assert.True(t, GetRegistry().TryLatchTouch(tenB, 42, 1000))
}
