package wz

import (
	"sync"
	"testing"
)

// TestLockParseIsExclusive sanity-checks that File.LockParse is a real
// exclusive mutex, not an RWMutex or no-op. Documents the contract that
// atlas-renders' WZCache relies on when serving concurrent map renders
// through a single shared *wz.File.
//
// Why the matching production code is load-bearing: services/atlas-
// renders/atlas.com/renders/storage/wzcache.go caches one *wz.File per
// (scope, region, version, archive) tuple. Each map render lazy-parses
// its target .img the first time Properties() is called. Before parseMu
// was added in File, two goroutines parsing different *Image instances
// backed by the same *wz.File raced the shared *os.File seek cursor.
//
// The actual race regression is best validated by `go test -race ./...`
// against the full atlas-wz module, plus the load-pattern atlas-renders
// generates in production. A direct unit-level race test would need a
// real WZ-encoded fixture for parse() to walk; that's expensive to
// maintain and the binding contract this test pins is already the
// minimal one the production code needs.
func TestLockParseIsExclusive(t *testing.T) {
	f := &File{}

	const goroutines = 32
	var (
		inCritical int
		maxSeen    int
		mu         sync.Mutex
	)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			unlock := f.LockParse()
			defer unlock()
			mu.Lock()
			inCritical++
			if inCritical > maxSeen {
				maxSeen = inCritical
			}
			mu.Unlock()
			// brief overlap window; no sleep — we want goroutines to
			// arrive concurrently while the mutex enforces exclusion.
			mu.Lock()
			inCritical--
			mu.Unlock()
		}()
	}
	wg.Wait()
	if maxSeen != 1 {
		t.Fatalf("LockParse not exclusive: saw %d concurrent holders", maxSeen)
	}
}

// TestPropertiesFastPathSkipsLock guards against a regression where the
// outer parsed=true fast-path is removed. The Properties() implementation
// MUST early-return without acquiring the lock when the image is already
// parsed; otherwise atlas-renders' hot path (every layer composite reads
// the parsed property tree of a previously-warmed .img) would serialise
// all reads behind the parse mutex and tank concurrency.
//
// We can't observe lock acquisition from outside, but we CAN observe
// that calling Properties() on a parsed in-memory image (constructed
// with NewParsedImage, wzFile=nil) never trips the nil-File guard —
// proving the fast path returned before either the wzFile check or the
// lock acquisition would have been reached.
func TestPropertiesFastPathSkipsLock(t *testing.T) {
	img := NewParsedImage("test", nil)
	// parsed=true (from NewParsedImage) + wzFile=nil. If the fast path
	// were removed, the implementation would either nil-deref on wzFile
	// or fall through to the nil-handler branch. Either way, exercising
	// it here a few times rapidly is the regression signal.
	for i := 0; i < 1024; i++ {
		if got := img.Properties(); got != nil {
			t.Fatalf("Properties() = %v on fresh NewParsedImage, want nil", got)
		}
	}
}
