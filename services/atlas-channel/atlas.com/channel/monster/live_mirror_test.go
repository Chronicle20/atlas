package monster

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// newTestLiveMirror constructs an isolated mirror (bypasses the singleton +
// sweeper goroutine), mirroring the newTestStatusMirror pattern.
func newTestLiveMirror() *LiveMirror {
	return &LiveMirror{perTenant: map[uuid.UUID]map[uint32]LiveEntry{}}
}

func testField() field.Model {
	return field.NewBuilder(0, 1, 100000000).Build()
}

func TestLiveMirror_PutLookupRoundTrip(t *testing.T) {
	m := newTestLiveMirror()
	tm := newTestTenant(t)
	e := LiveEntry{Field: testField(), MonsterId: 100100, Mp: 50, MaxMp: 80, ControllerHasAggro: true}
	m.Put(tm, 7, e)

	got, ok := m.Lookup(tm, 7)
	if !ok {
		t.Fatalf("expected hit after Put")
	}
	if got.MonsterId != 100100 || got.Mp != 50 || got.MaxMp != 80 || !got.ControllerHasAggro {
		t.Fatalf("entry mismatch: %+v", got)
	}
	if got.LastWrite.IsZero() {
		t.Fatalf("Put must stamp LastWrite")
	}
}

func TestLiveMirror_LookupMiss(t *testing.T) {
	m := newTestLiveMirror()
	tm := newTestTenant(t)
	if _, ok := m.Lookup(tm, 999); ok {
		t.Fatalf("expected miss on empty mirror")
	}
}

func TestLiveMirror_UpdateMp_NoOpWhenAbsent(t *testing.T) {
	m := newTestLiveMirror()
	tm := newTestTenant(t)
	m.UpdateMp(tm, 7, 42)
	if _, ok := m.Lookup(tm, 7); ok {
		t.Fatalf("UpdateMp must never create an entry")
	}
}

func TestLiveMirror_UpdateAggro_NoOpWhenAbsent(t *testing.T) {
	m := newTestLiveMirror()
	tm := newTestTenant(t)
	m.UpdateAggro(tm, 7, true)
	if _, ok := m.Lookup(tm, 7); ok {
		t.Fatalf("UpdateAggro must never create an entry")
	}
}

func TestLiveMirror_UpdateMpAndAggro_MutatePresentEntry(t *testing.T) {
	m := newTestLiveMirror()
	tm := newTestTenant(t)
	m.Put(tm, 7, LiveEntry{Field: testField(), MonsterId: 100100, Mp: 50, MaxMp: 80})
	before, _ := m.Lookup(tm, 7)

	time.Sleep(time.Millisecond)
	m.UpdateMp(tm, 7, 12)
	m.UpdateAggro(tm, 7, true)

	got, ok := m.Lookup(tm, 7)
	if !ok {
		t.Fatalf("expected hit")
	}
	if got.Mp != 12 || !got.ControllerHasAggro {
		t.Fatalf("updates not applied: %+v", got)
	}
	if got.MonsterId != 100100 || got.MaxMp != 80 {
		t.Fatalf("updates must not clobber other fields: %+v", got)
	}
	if !got.LastWrite.After(before.LastWrite) {
		t.Fatalf("every write must refresh LastWrite")
	}
}

func TestLiveMirror_Remove(t *testing.T) {
	m := newTestLiveMirror()
	tm := newTestTenant(t)
	m.Put(tm, 7, LiveEntry{Field: testField(), MonsterId: 100100})
	m.Remove(tm, 7)
	if _, ok := m.Lookup(tm, 7); ok {
		t.Fatalf("expected miss after Remove")
	}
}

func TestLiveMirror_TenantIsolationAndEviction(t *testing.T) {
	m := newTestLiveMirror()
	t1 := newTestTenant(t)
	t2 := newTestTenant(t)
	m.Put(t1, 7, LiveEntry{Field: testField(), MonsterId: 111})
	m.Put(t2, 7, LiveEntry{Field: testField(), MonsterId: 222})

	got, _ := m.Lookup(t1, 7)
	if got.MonsterId != 111 {
		t.Fatalf("cross-tenant bleed: %+v", got)
	}

	m.EvictTenant(t1.Id())
	if _, ok := m.Lookup(t1, 7); ok {
		t.Fatalf("expected t1 evicted")
	}
	if _, ok := m.Lookup(t2, 7); !ok {
		t.Fatalf("t2 must survive t1 eviction")
	}
}

func TestLiveMirror_SweepStale(t *testing.T) {
	m := newTestLiveMirror()
	tm := newTestTenant(t)
	m.Put(tm, 1, LiveEntry{Field: testField(), MonsterId: 1})
	m.Put(tm, 2, LiveEntry{Field: testField(), MonsterId: 2})

	// Drive staleness with a synthetic "now" 31m ahead of both LastWrites.
	future := time.Now().Add(31 * time.Minute)
	evicted := m.SweepStale(future, 30*time.Minute)
	if evicted != 2 {
		t.Fatalf("expected both entries stale at now+31m, got %d", evicted)
	}

	m.Put(tm, 3, LiveEntry{Field: testField(), MonsterId: 3})
	evicted = m.SweepStale(time.Now(), 30*time.Minute)
	if evicted != 0 {
		t.Fatalf("fresh entry must survive, evicted %d", evicted)
	}
	if _, ok := m.Lookup(tm, 3); !ok {
		t.Fatalf("fresh entry must still be present")
	}
}

func TestLiveMirror_ConcurrentAccess(t *testing.T) {
	m := newTestLiveMirror()
	tm := newTestTenant(t)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				id := uint32(j % 10)
				m.Put(tm, id, LiveEntry{Field: testField(), MonsterId: id})
				m.UpdateMp(tm, id, uint32(j))
				m.UpdateAggro(tm, id, j%2 == 0)
				m.Lookup(tm, id)
				if j%50 == 0 {
					m.Remove(tm, id)
					m.SweepStale(time.Now(), time.Minute)
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestLiveEntryFromModel_MapsAllFields(t *testing.T) {
	f := testField()
	mo, err := NewModelBuilder(7, f, 100100).
		SetMp(33).
		SetMaxMp(90).
		SetControllerHasAggro(true).
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	e := LiveEntryFromModel(mo)
	if e.Field.WorldId() != f.WorldId() || e.Field.ChannelId() != f.ChannelId() || e.Field.MapId() != f.MapId() {
		t.Fatalf("field mismatch: %+v", e.Field)
	}
	if e.MonsterId != 100100 || e.Mp != 33 || e.MaxMp != 90 || !e.ControllerHasAggro {
		t.Fatalf("entry mismatch: %+v", e)
	}
}
