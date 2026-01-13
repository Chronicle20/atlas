package chalkboard

import (
	"sync"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

func newTestTenant() tenant.Model {
	t, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return t
}

func TestRegistryGetNonExistent(t *testing.T) {
	r := &Registry{characterRegister: make(map[ChalkboardKey]string)}
	ten := newTestTenant()

	_, ok := r.Get(ten, 12345)
	if ok {
		t.Error("Expected Get to return false for non-existent key")
	}
}

func TestRegistrySetAndGet(t *testing.T) {
	r := &Registry{characterRegister: make(map[ChalkboardKey]string)}
	ten := newTestTenant()
	characterId := uint32(12345)
	message := "Hello, World!"

	r.Set(ten, characterId, message)

	got, ok := r.Get(ten, characterId)
	if !ok {
		t.Fatal("Expected Get to return true for existing key")
	}
	if got != message {
		t.Errorf("Expected message %q, got %q", message, got)
	}
}

func TestRegistryClear(t *testing.T) {
	r := &Registry{characterRegister: make(map[ChalkboardKey]string)}
	ten := newTestTenant()
	characterId := uint32(12345)

	r.Set(ten, characterId, "test message")

	cleared := r.Clear(ten, characterId)
	if !cleared {
		t.Error("Expected Clear to return true for existing key")
	}

	_, ok := r.Get(ten, characterId)
	if ok {
		t.Error("Expected Get to return false after Clear")
	}
}

func TestRegistryClearNonExistent(t *testing.T) {
	r := &Registry{characterRegister: make(map[ChalkboardKey]string)}
	ten := newTestTenant()

	cleared := r.Clear(ten, 99999)
	if cleared {
		t.Error("Expected Clear to return false for non-existent key")
	}
}

func TestRegistryTenantIsolation(t *testing.T) {
	r := &Registry{characterRegister: make(map[ChalkboardKey]string)}
	tenant1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tenant2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)
	characterId := uint32(12345)

	r.Set(tenant1, characterId, "tenant1 message")
	r.Set(tenant2, characterId, "tenant2 message")

	msg1, ok1 := r.Get(tenant1, characterId)
	if !ok1 {
		t.Fatal("Expected Get to return true for tenant1")
	}
	if msg1 != "tenant1 message" {
		t.Errorf("Expected tenant1 message, got %q", msg1)
	}

	msg2, ok2 := r.Get(tenant2, characterId)
	if !ok2 {
		t.Fatal("Expected Get to return true for tenant2")
	}
	if msg2 != "tenant2 message" {
		t.Errorf("Expected tenant2 message, got %q", msg2)
	}

	r.Clear(tenant1, characterId)

	_, ok1After := r.Get(tenant1, characterId)
	if ok1After {
		t.Error("Expected tenant1 data to be cleared")
	}

	msg2After, ok2After := r.Get(tenant2, characterId)
	if !ok2After {
		t.Error("Expected tenant2 data to still exist after clearing tenant1")
	}
	if msg2After != "tenant2 message" {
		t.Errorf("Expected tenant2 message to remain unchanged, got %q", msg2After)
	}
}

func TestRegistryConcurrentAccess(t *testing.T) {
	r := &Registry{characterRegister: make(map[ChalkboardKey]string)}
	ten := newTestTenant()

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			characterId := uint32(id)
			r.Set(ten, characterId, "message")
			r.Get(ten, characterId)
			r.Clear(ten, characterId)
		}(i)
	}

	wg.Wait()
}

func TestRegistryOverwrite(t *testing.T) {
	r := &Registry{characterRegister: make(map[ChalkboardKey]string)}
	ten := newTestTenant()
	characterId := uint32(12345)

	r.Set(ten, characterId, "first message")
	r.Set(ten, characterId, "second message")

	got, ok := r.Get(ten, characterId)
	if !ok {
		t.Fatal("Expected Get to return true")
	}
	if got != "second message" {
		t.Errorf("Expected overwritten message, got %q", got)
	}
}
