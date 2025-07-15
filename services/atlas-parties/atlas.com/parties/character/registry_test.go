package character

import (
	"testing"

	"github.com/Chronicle20/atlas-constants/job"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRegistryCreate(t *testing.T) {
	r := GetRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	
	characterId := uint32(123)
	name := "TestChar"
	level := byte(10)
	jobId := job.Id(100)
	worldId := byte(1)
	channelId := byte(1)
	mapId := uint32(100000)
	gm := 0
	
	char := r.Create(ten, worldId, channelId, mapId, characterId, name, level, jobId, gm)
	
	assert.Equal(t, characterId, char.Id())
	assert.Equal(t, name, char.Name())
	assert.Equal(t, level, char.Level())
	assert.Equal(t, jobId, char.JobId())
	assert.Equal(t, worldId, char.WorldId())
	assert.Equal(t, channelId, char.ChannelId())
	assert.Equal(t, mapId, char.MapId())
	assert.Equal(t, uint32(0), char.PartyId())
	assert.Equal(t, false, char.Online())
	assert.Equal(t, gm, char.GM())
	
	// Cleanup
	r.Delete(ten, characterId)
}

func TestRegistryGet(t *testing.T) {
	r := GetRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	
	characterId := uint32(124)
	name := "TestChar2"
	level := byte(15)
	jobId := job.Id(200)
	
	// Create character
	created := r.Create(ten, 1, 1, 100000, characterId, name, level, jobId, 0)
	
	// Get character
	retrieved, err := r.Get(ten, characterId)
	assert.NoError(t, err)
	assert.Equal(t, created.Id(), retrieved.Id())
	assert.Equal(t, created.Name(), retrieved.Name())
	assert.Equal(t, created.Level(), retrieved.Level())
	assert.Equal(t, created.JobId(), retrieved.JobId())
	
	// Cleanup
	r.Delete(ten, characterId)
}

func TestRegistryGetNotFound(t *testing.T) {
	r := GetRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	
	nonExistentId := uint32(999)
	
	_, err := r.Get(ten, nonExistentId)
	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
}

func TestRegistryUpdateLevelChange(t *testing.T) {
	r := GetRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	
	characterId := uint32(125)
	initialLevel := byte(10)
	newLevel := byte(15)
	
	// Create character
	r.Create(ten, 1, 1, 100000, characterId, "TestChar", initialLevel, job.Id(100), 0)
	
	// Update level
	updated := r.Update(ten, characterId, func(m Model) Model {
		return m.ChangeLevel(newLevel)
	})
	
	assert.Equal(t, newLevel, updated.Level())
	
	// Verify persistence
	retrieved, err := r.Get(ten, characterId)
	assert.NoError(t, err)
	assert.Equal(t, newLevel, retrieved.Level())
	
	// Cleanup
	r.Delete(ten, characterId)
}

func TestRegistryUpdateJobChange(t *testing.T) {
	r := GetRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	
	characterId := uint32(126)
	initialJobId := job.Id(100)
	newJobId := job.Id(200)
	
	// Create character
	r.Create(ten, 1, 1, 100000, characterId, "TestChar", 10, initialJobId, 0)
	
	// Update job
	updated := r.Update(ten, characterId, func(m Model) Model {
		return m.ChangeJob(newJobId)
	})
	
	assert.Equal(t, newJobId, updated.JobId())
	
	// Verify persistence
	retrieved, err := r.Get(ten, characterId)
	assert.NoError(t, err)
	assert.Equal(t, newJobId, retrieved.JobId())
	
	// Cleanup
	r.Delete(ten, characterId)
}

func TestRegistryUpdatePartyOperations(t *testing.T) {
	r := GetRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	
	characterId := uint32(127)
	partyId := uint32(456)
	
	// Create character
	r.Create(ten, 1, 1, 100000, characterId, "TestChar", 10, job.Id(100), 0)
	
	// Join party
	updated := r.Update(ten, characterId, func(m Model) Model {
		return m.JoinParty(partyId)
	})
	
	assert.Equal(t, partyId, updated.PartyId())
	
	// Verify persistence
	retrieved, err := r.Get(ten, characterId)
	assert.NoError(t, err)
	assert.Equal(t, partyId, retrieved.PartyId())
	
	// Leave party
	updated = r.Update(ten, characterId, func(m Model) Model {
		return m.LeaveParty()
	})
	
	assert.Equal(t, uint32(0), updated.PartyId())
	
	// Verify persistence
	retrieved, err = r.Get(ten, characterId)
	assert.NoError(t, err)
	assert.Equal(t, uint32(0), retrieved.PartyId())
	
	// Cleanup
	r.Delete(ten, characterId)
}

func TestRegistryUpdateLoginLogout(t *testing.T) {
	r := GetRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	
	characterId := uint32(128)
	
	// Create character (initially offline)
	r.Create(ten, 1, 1, 100000, characterId, "TestChar", 10, job.Id(100), 0)
	
	// Login
	updated := r.Update(ten, characterId, func(m Model) Model {
		return m.Login()
	})
	
	assert.Equal(t, true, updated.Online())
	
	// Verify persistence
	retrieved, err := r.Get(ten, characterId)
	assert.NoError(t, err)
	assert.Equal(t, true, retrieved.Online())
	
	// Logout
	updated = r.Update(ten, characterId, func(m Model) Model {
		return m.Logout()
	})
	
	assert.Equal(t, false, updated.Online())
	
	// Verify persistence
	retrieved, err = r.Get(ten, characterId)
	assert.NoError(t, err)
	assert.Equal(t, false, retrieved.Online())
	
	// Cleanup
	r.Delete(ten, characterId)
}

func TestRegistryUpdateMapChange(t *testing.T) {
	r := GetRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	
	characterId := uint32(129)
	initialMapId := uint32(100000)
	newMapId := uint32(200000)
	
	// Create character
	r.Create(ten, 1, 1, initialMapId, characterId, "TestChar", 10, job.Id(100), 0)
	
	// Change map
	updated := r.Update(ten, characterId, func(m Model) Model {
		return m.ChangeMap(newMapId)
	})
	
	assert.Equal(t, newMapId, updated.MapId())
	
	// Verify persistence
	retrieved, err := r.Get(ten, characterId)
	assert.NoError(t, err)
	assert.Equal(t, newMapId, retrieved.MapId())
	
	// Cleanup
	r.Delete(ten, characterId)
}

func TestRegistryUpdateChannelChange(t *testing.T) {
	r := GetRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	
	characterId := uint32(130)
	initialChannelId := byte(1)
	newChannelId := byte(2)
	
	// Create character
	r.Create(ten, 1, initialChannelId, 100000, characterId, "TestChar", 10, job.Id(100), 0)
	
	// Change channel
	updated := r.Update(ten, characterId, func(m Model) Model {
		return m.ChangeChannel(newChannelId)
	})
	
	assert.Equal(t, newChannelId, updated.ChannelId())
	
	// Verify persistence
	retrieved, err := r.Get(ten, characterId)
	assert.NoError(t, err)
	assert.Equal(t, newChannelId, retrieved.ChannelId())
	
	// Cleanup
	r.Delete(ten, characterId)
}

func TestRegistryUpdateMultipleUpdaters(t *testing.T) {
	r := GetRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	
	characterId := uint32(131)
	initialLevel := byte(10)
	newLevel := byte(15)
	newJobId := job.Id(200)
	partyId := uint32(456)
	
	// Create character
	r.Create(ten, 1, 1, 100000, characterId, "TestChar", initialLevel, job.Id(100), 0)
	
	// Apply multiple updates in one call
	updated := r.Update(ten, characterId, 
		func(m Model) Model { return m.ChangeLevel(newLevel) },
		func(m Model) Model { return m.ChangeJob(newJobId) },
		func(m Model) Model { return m.JoinParty(partyId) },
		func(m Model) Model { return m.Login() },
	)
	
	assert.Equal(t, newLevel, updated.Level())
	assert.Equal(t, newJobId, updated.JobId())
	assert.Equal(t, partyId, updated.PartyId())
	assert.Equal(t, true, updated.Online())
	
	// Verify persistence
	retrieved, err := r.Get(ten, characterId)
	assert.NoError(t, err)
	assert.Equal(t, newLevel, retrieved.Level())
	assert.Equal(t, newJobId, retrieved.JobId())
	assert.Equal(t, partyId, retrieved.PartyId())
	assert.Equal(t, true, retrieved.Online())
	
	// Cleanup
	r.Delete(ten, characterId)
}

func TestRegistryDelete(t *testing.T) {
	r := GetRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	
	characterId := uint32(132)
	
	// Create character
	r.Create(ten, 1, 1, 100000, characterId, "TestChar", 10, job.Id(100), 0)
	
	// Verify character exists
	_, err := r.Get(ten, characterId)
	assert.NoError(t, err)
	
	// Delete character
	err = r.Delete(ten, characterId)
	assert.NoError(t, err)
	
	// Verify character is deleted
	_, err = r.Get(ten, characterId)
	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
}

func TestRegistryDeleteNotFound(t *testing.T) {
	r := GetRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	
	nonExistentId := uint32(999)
	
	err := r.Delete(ten, nonExistentId)
	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
}

func TestRegistryMultiTenantIsolation(t *testing.T) {
	r := GetRegistry()
	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "GMS", 87, 1)
	
	characterId := uint32(133)
	
	// Create character in tenant 1
	r.Create(ten1, 1, 1, 100000, characterId, "TestChar1", 10, job.Id(100), 0)
	
	// Create character with same ID in tenant 2
	r.Create(ten2, 1, 1, 100000, characterId, "TestChar2", 20, job.Id(200), 0)
	
	// Verify isolation
	char1, err := r.Get(ten1, characterId)
	assert.NoError(t, err)
	assert.Equal(t, "TestChar1", char1.Name())
	assert.Equal(t, byte(10), char1.Level())
	assert.Equal(t, job.Id(100), char1.JobId())
	
	char2, err := r.Get(ten2, characterId)
	assert.NoError(t, err)
	assert.Equal(t, "TestChar2", char2.Name())
	assert.Equal(t, byte(20), char2.Level())
	assert.Equal(t, job.Id(200), char2.JobId())
	
	// Update in tenant 1 should not affect tenant 2
	r.Update(ten1, characterId, func(m Model) Model {
		return m.ChangeLevel(25)
	})
	
	char1, _ = r.Get(ten1, characterId)
	char2, _ = r.Get(ten2, characterId)
	
	assert.Equal(t, byte(25), char1.Level())
	assert.Equal(t, byte(20), char2.Level())
	
	// Cleanup
	r.Delete(ten1, characterId)
	r.Delete(ten2, characterId)
}

func TestRegistryUpdateOrderMatters(t *testing.T) {
	r := GetRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	
	characterId := uint32(134)
	
	// Create character
	r.Create(ten, 1, 1, 100000, characterId, "TestChar", 10, job.Id(100), 0)
	
	// Test that updaters are applied in order
	updated := r.Update(ten, characterId, 
		func(m Model) Model { return m.ChangeLevel(15) },
		func(m Model) Model { return m.ChangeLevel(20) },
		func(m Model) Model { return m.ChangeLevel(25) },
	)
	
	// Final level should be 25 (last updater wins)
	assert.Equal(t, byte(25), updated.Level())
	
	// Cleanup
	r.Delete(ten, characterId)
}

func TestRegistryUpdateEmptyUpdaters(t *testing.T) {
	r := GetRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	
	characterId := uint32(135)
	originalLevel := byte(10)
	
	// Create character
	original := r.Create(ten, 1, 1, 100000, characterId, "TestChar", originalLevel, job.Id(100), 0)
	
	// Update with no updaters
	updated := r.Update(ten, characterId)
	
	// Should return unchanged character
	assert.Equal(t, original.Level(), updated.Level())
	assert.Equal(t, original.JobId(), updated.JobId())
	assert.Equal(t, original.PartyId(), updated.PartyId())
	assert.Equal(t, original.Online(), updated.Online())
	
	// Cleanup
	r.Delete(ten, characterId)
}