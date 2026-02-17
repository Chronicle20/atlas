package instance

import (
	"sync"
	"testing"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return ten
}

func testModel(characters ...CharacterEntry) Model {
	m, _ := NewBuilder().
		SetTenantId(uuid.New()).
		SetDefinitionId(uuid.New()).
		SetQuestId("test_pq").
		SetCharacters(characters).
		Build()
	return m
}

func TestRegistry_CreateAndGet(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := testTenant(t)

	m := testModel(NewCharacterEntry(1000, 0, 0))
	r.Create(ten, m)

	got, err := r.Get(ten, m.Id())
	require.NoError(t, err)
	assert.Equal(t, m.Id(), got.Id())
	assert.Equal(t, "test_pq", got.QuestId())
}

func TestRegistry_Get_NotFound(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := testTenant(t)

	_, err := r.Get(ten, uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRegistry_GetByCharacter(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := testTenant(t)

	chars := []CharacterEntry{
		NewCharacterEntry(2000, 0, 0),
		NewCharacterEntry(2001, 0, 0),
	}
	m := testModel(chars...)
	r.Create(ten, m)

	for _, c := range chars {
		got, err := r.GetByCharacter(ten, c.CharacterId())
		require.NoError(t, err)
		assert.Equal(t, m.Id(), got.Id())
	}

	_, err := r.GetByCharacter(ten, 9999)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRegistry_GetAll(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := testTenant(t)

	m1 := testModel(NewCharacterEntry(3000, 0, 0))
	m2 := testModel(NewCharacterEntry(3001, 0, 0))
	r.Create(ten, m1)
	r.Create(ten, m2)

	all := r.GetAll(ten)
	assert.Len(t, all, 2)
}

func TestRegistry_Update(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := testTenant(t)

	m := testModel(NewCharacterEntry(4000, 0, 0))
	r.Create(ten, m)

	updated, err := r.Update(ten, m.Id(), func(m Model) Model {
		return m.SetState(StateActive)
	})
	require.NoError(t, err)
	assert.Equal(t, StateActive, updated.State())

	got, err := r.Get(ten, m.Id())
	require.NoError(t, err)
	assert.Equal(t, StateActive, got.State())
}

func TestRegistry_Update_NotFound(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := testTenant(t)

	_, err := r.Update(ten, uuid.New(), func(m Model) Model {
		return m.SetState(StateActive)
	})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRegistry_Update_CharacterIndex(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := testTenant(t)

	m := testModel(NewCharacterEntry(5000, 0, 0))
	r.Create(ten, m)

	// Add a character
	_, err := r.Update(ten, m.Id(), func(m Model) Model {
		return m.AddCharacter(NewCharacterEntry(5001, 0, 0))
	})
	require.NoError(t, err)

	got, err := r.GetByCharacter(ten, 5001)
	require.NoError(t, err)
	assert.Equal(t, m.Id(), got.Id())

	// Remove the original character
	_, err = r.Update(ten, m.Id(), func(m Model) Model {
		return m.RemoveCharacter(5000)
	})
	require.NoError(t, err)

	_, err = r.GetByCharacter(ten, 5000)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRegistry_Remove(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := testTenant(t)

	m := testModel(NewCharacterEntry(6000, 0, 0))
	r.Create(ten, m)

	r.Remove(ten, m.Id())

	_, err := r.Get(ten, m.Id())
	assert.ErrorIs(t, err, ErrNotFound)

	_, err = r.GetByCharacter(ten, 6000)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRegistry_Clear(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := testTenant(t)

	r.Create(ten, testModel(NewCharacterEntry(7000, 0, 0)))
	r.Create(ten, testModel(NewCharacterEntry(7001, 0, 0)))
	assert.Len(t, r.GetAll(ten), 2)

	r.Clear(ten)
	assert.Len(t, r.GetAll(ten), 0)
}

func TestRegistry_TenantIsolation(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()

	ten1 := testTenant(t)
	ten2 := testTenant(t)

	m := testModel(NewCharacterEntry(8000, 0, 0))
	r.Create(ten1, m)

	got, err := r.Get(ten1, m.Id())
	require.NoError(t, err)
	assert.Equal(t, m.Id(), got.Id())

	_, err = r.Get(ten2, m.Id())
	assert.ErrorIs(t, err, ErrNotFound)

	_, err = r.GetByCharacter(ten2, 8000)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := testTenant(t)

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent creates
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			m := testModel(NewCharacterEntry(uint32(10000+idx), 0, 0))
			r.Create(ten, m)
		}(i)
	}
	wg.Wait()

	all := r.GetAll(ten)
	assert.Len(t, all, iterations)

	// Concurrent reads
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = r.GetByCharacter(ten, uint32(10000+idx))
		}(i)
	}
	wg.Wait()
}

func TestRegistry_LeaveWorkflow(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := testTenant(t)

	chars := []CharacterEntry{
		NewCharacterEntry(11000, 0, 0),
		NewCharacterEntry(11001, 0, 0),
		NewCharacterEntry(11002, 0, 0),
	}
	m := testModel(chars...)
	m = m.SetState(StateActive)
	r.Create(ten, m)

	// Leave: remove one character
	updated, err := r.Update(ten, m.Id(), func(m Model) Model {
		return m.RemoveCharacter(11001)
	})
	require.NoError(t, err)
	assert.Len(t, updated.Characters(), 2)

	// Removed character should not be in index
	_, err = r.GetByCharacter(ten, 11001)
	assert.ErrorIs(t, err, ErrNotFound)

	// Remaining characters should still resolve
	got, err := r.GetByCharacter(ten, 11000)
	require.NoError(t, err)
	assert.Equal(t, m.Id(), got.Id())

	got, err = r.GetByCharacter(ten, 11002)
	require.NoError(t, err)
	assert.Equal(t, m.Id(), got.Id())
}

func TestRegistry_LeaveLastCharacter(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()
	ten := testTenant(t)

	m := testModel(NewCharacterEntry(12000, 0, 0))
	m = m.SetState(StateActive)
	r.Create(ten, m)

	// Remove the only character
	updated, err := r.Update(ten, m.Id(), func(m Model) Model {
		return m.RemoveCharacter(12000)
	})
	require.NoError(t, err)
	assert.Len(t, updated.Characters(), 0)

	// Character index should be cleaned up
	_, err = r.GetByCharacter(ten, 12000)
	assert.ErrorIs(t, err, ErrNotFound)

	// Instance still exists (Destroy is separate)
	_, err = r.Get(ten, m.Id())
	require.NoError(t, err)

	// Simulate Destroy after empty check
	r.Remove(ten, m.Id())
	_, err = r.Get(ten, m.Id())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRegistry_ConcurrentMultipleTenants(t *testing.T) {
	r := GetRegistry()
	r.ResetForTesting()

	ten1 := testTenant(t)
	ten2 := testTenant(t)

	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			m := testModel(NewCharacterEntry(uint32(20000+idx), 0, 0))
			r.Create(ten1, m)
		}(i)
		go func(idx int) {
			defer wg.Done()
			m := testModel(NewCharacterEntry(uint32(30000+idx), 0, 0))
			r.Create(ten2, m)
		}(i)
	}
	wg.Wait()

	assert.Len(t, r.GetAll(ten1), 50)
	assert.Len(t, r.GetAll(ten2), 50)
}
