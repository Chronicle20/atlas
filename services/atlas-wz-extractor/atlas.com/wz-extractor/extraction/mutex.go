package extraction

import "sync"

type tenantMutexRegistry struct {
	mu sync.Mutex
	m  map[string]*sync.Mutex
}

var tenantMu = &tenantMutexRegistry{m: make(map[string]*sync.Mutex)}

func (r *tenantMutexRegistry) get(key string) *sync.Mutex {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m, ok := r.m[key]; ok {
		return m
	}
	m := &sync.Mutex{}
	r.m[key] = m
	return m
}

// Acquire blocks until the per-key mutex is held.
func Acquire(key string) *sync.Mutex {
	m := tenantMu.get(key)
	m.Lock()
	return m
}

// TryAcquire returns the mutex (locked) and true on success, or (nil, false) when the mutex is already held.
func TryAcquire(key string) (*sync.Mutex, bool) {
	m := tenantMu.get(key)
	if m.TryLock() {
		return m, true
	}
	return nil, false
}

// Release unlocks a previously acquired mutex.
func Release(m *sync.Mutex) {
	if m == nil {
		return
	}
	m.Unlock()
}
