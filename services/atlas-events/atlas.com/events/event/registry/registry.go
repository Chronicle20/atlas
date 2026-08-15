package registry

import "sync"

var (
	mu       sync.RWMutex
	handlers = map[string]Handler{}
)

// Register makes h the handler for h.Type(). Called once per event package from
// main.go. Duplicate registration is a programming error, not a runtime
// condition, so it panics at startup rather than silently shadowing.
func Register(h Handler) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := handlers[h.Type()]; exists {
		panic("event registry: duplicate handler for type " + h.Type())
	}
	handlers[h.Type()] = h
}

// Get resolves a handler by definition type. A false second return is a
// FAILURE the caller must surface (the work row fails with
// "no handler for type X"), never a fallback to a default handler.
func Get(theType string) (Handler, bool) {
	mu.RLock()
	defer mu.RUnlock()
	h, ok := handlers[theType]
	return h, ok
}

// Types lists every registered type. Used by the definition REST layer to
// reject an unknown type at write time rather than at trigger time (FR-D6).
func Types() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(handlers))
	for k := range handlers {
		out = append(out, k)
	}
	return out
}

// reset clears the registry. Test-only; the production path registers once at
// startup.
func reset() {
	mu.Lock()
	defer mu.Unlock()
	handlers = map[string]Handler{}
}
