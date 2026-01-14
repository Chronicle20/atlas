package chair

import (
	"sync"
	"testing"
)

func resetRegistry() {
	r := GetRegistry()
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.characterRegister = make(map[uint32]Model)
}

func TestRegistry_GetSet(t *testing.T) {
	resetRegistry()

	characterId := uint32(12345)
	m := Model{id: 1, chairType: "FIXED"}

	// Initially should not exist
	_, ok := GetRegistry().Get(characterId)
	if ok {
		t.Fatal("Expected character to not exist in registry initially")
	}

	// Set the chair
	GetRegistry().Set(characterId, m)

	// Now should exist
	retrieved, ok := GetRegistry().Get(characterId)
	if !ok {
		t.Fatal("Expected character to exist in registry after Set")
	}

	if retrieved.Id() != m.Id() {
		t.Errorf("Id mismatch. Expected %d, got %d", m.Id(), retrieved.Id())
	}

	if retrieved.Type() != m.Type() {
		t.Errorf("Type mismatch. Expected %s, got %s", m.Type(), retrieved.Type())
	}
}

func TestRegistry_Clear(t *testing.T) {
	resetRegistry()

	characterId := uint32(12345)
	m := Model{id: 1, chairType: "FIXED"}

	// Set the chair
	GetRegistry().Set(characterId, m)

	// Verify it exists
	_, ok := GetRegistry().Get(characterId)
	if !ok {
		t.Fatal("Expected character to exist in registry after Set")
	}

	// Clear should return true
	existed := GetRegistry().Clear(characterId)
	if !existed {
		t.Fatal("Expected Clear to return true for existing entry")
	}

	// Now should not exist
	_, ok = GetRegistry().Get(characterId)
	if ok {
		t.Fatal("Expected character to not exist in registry after Clear")
	}
}

func TestRegistry_Clear_NotExists(t *testing.T) {
	resetRegistry()

	characterId := uint32(99999)

	// Clear non-existent entry should return false
	existed := GetRegistry().Clear(characterId)
	if existed {
		t.Fatal("Expected Clear to return false for non-existent entry")
	}
}

func TestRegistry_Concurrent(t *testing.T) {
	resetRegistry()

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent writes
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			characterId := uint32(id)
			m := Model{id: uint32(id), chairType: "FIXED"}
			GetRegistry().Set(characterId, m)
		}(i)
	}

	wg.Wait()

	// Verify all entries exist
	for i := 0; i < iterations; i++ {
		characterId := uint32(i)
		m, ok := GetRegistry().Get(characterId)
		if !ok {
			t.Errorf("Expected character %d to exist in registry", characterId)
			continue
		}
		if m.Id() != uint32(i) {
			t.Errorf("Expected chair id %d, got %d", i, m.Id())
		}
	}

	// Concurrent reads and clears
	for i := 0; i < iterations; i++ {
		wg.Add(2)
		go func(id int) {
			defer wg.Done()
			characterId := uint32(id)
			GetRegistry().Get(characterId)
		}(i)
		go func(id int) {
			defer wg.Done()
			characterId := uint32(id)
			GetRegistry().Clear(characterId)
		}(i)
	}

	wg.Wait()
}

func TestRegistry_MultipleCharacters(t *testing.T) {
	resetRegistry()

	// Set chairs for multiple characters
	chars := []struct {
		characterId uint32
		chairId     uint32
		chairType   string
	}{
		{100, 0, "FIXED"},
		{200, 3010001, "PORTABLE"},
		{300, 1, "FIXED"},
	}

	for _, c := range chars {
		m := Model{id: c.chairId, chairType: c.chairType}
		GetRegistry().Set(c.characterId, m)
	}

	// Verify all entries
	for _, c := range chars {
		m, ok := GetRegistry().Get(c.characterId)
		if !ok {
			t.Errorf("Expected character %d to exist", c.characterId)
			continue
		}
		if m.Id() != c.chairId {
			t.Errorf("Character %d: expected chair id %d, got %d", c.characterId, c.chairId, m.Id())
		}
		if m.Type() != c.chairType {
			t.Errorf("Character %d: expected chair type %s, got %s", c.characterId, c.chairType, m.Type())
		}
	}

	// Clear one and verify others unaffected
	GetRegistry().Clear(200)

	_, ok := GetRegistry().Get(200)
	if ok {
		t.Error("Expected character 200 to be cleared")
	}

	_, ok = GetRegistry().Get(100)
	if !ok {
		t.Error("Expected character 100 to still exist")
	}

	_, ok = GetRegistry().Get(300)
	if !ok {
		t.Error("Expected character 300 to still exist")
	}
}
