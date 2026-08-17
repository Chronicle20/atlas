package character

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestRegistryUpdateNameChange covers ChangeName, the mutator the NAME_CHANGED
// consumer drives. The registry copy of the name is what atlas-channel renders
// in the party window (its toPartyMembers reads m.Name()), so a name that does
// not move here is a name that never reaches the client.
func TestRegistryUpdateNameChange(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := createTestCtx(ten)

	characterId := uint32(4227)
	f := field.NewBuilder(1, 1, 100000).Build()
	r.Create(ctx, f, characterId, "OldName", 30, job.Id(110), 0)

	updated := r.Update(ctx, characterId, func(m Model) Model {
		return m.ChangeName("NewName")
	})
	assert.Equal(t, "NewName", updated.Name())

	retrieved, err := r.Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Equal(t, "NewName", retrieved.Name())

	// The rename must not disturb the rest of the model — ChangeName rebuilds
	// the struct field by field, so a dropped field would silently zero here.
	assert.Equal(t, byte(30), retrieved.Level())
	assert.Equal(t, job.Id(110), retrieved.JobId())
	assert.Equal(t, characterId, retrieved.Id())
}
