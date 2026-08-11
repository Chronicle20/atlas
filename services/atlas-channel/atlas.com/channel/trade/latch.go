package trade

import (
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// CreateLatch orders a trade INVITE behind the CREATE_ROOM it belongs to.
//
// For roomType 3 the reference client opens a trade in TWO sends — mode 0
// (create) then mode 2 (invite) — and atlas-socket dispatches ONE GOROUTINE PER
// INBOUND PACKET (libs/atlas-socket/server.go, the read loop's
// `routine.Go(... handle ...)`). Nothing serialises the two: the create arm
// runs three sequential REST occupancy probes before it produces its command,
// while the invite arm produces immediately. Both commands are keyed by the
// acting character on COMMAND_TOPIC_TRADE, so consume order IS produce order —
// and a client whose two sends are closer together than the create's probe
// latency has its invite overtake its create. atlas-trades then answers an
// invite from a character with no room, which is the `UNABLE` refusal observed
// in testing: the two clients differed only in how far apart they paced the
// sends (~50 ms worked, ~2 ms did not).
//
// The latch is armed by the create arm before it does any work and released
// once CREATE_ROOM has been produced (or refused). The invite arm waits on it,
// so the ordering the client's send order implies is actually enforced.
//
// It deliberately does NOT track rooms — a released latch means the command is
// on the topic ahead of the invite, which is the whole invariant. Room state
// stays authoritative in atlas-trades.
type CreateLatch struct {
	mutex    sync.Mutex
	inFlight map[latchKey]chan struct{}
}

type latchKey struct {
	tenant      tenant.Model
	characterId character.Id
}

var (
	createLatch     *CreateLatch
	createLatchOnce sync.Once
)

func GetCreateLatch() *CreateLatch {
	createLatchOnce.Do(func() {
		createLatch = &CreateLatch{inFlight: make(map[latchKey]chan struct{})}
	})
	return createLatch
}

// Begin arms the latch for one character and returns the release. A second
// concurrent create for the same character does not disturb the first — it gets
// a no-op release, so the owner of the latch is the only one who can clear it
// and a nested release cannot open the gate early.
func (r *CreateLatch) Begin(t tenant.Model, characterId character.Id) func() {
	k := latchKey{tenant: t, characterId: characterId}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, ok := r.inFlight[k]; ok {
		return func() {}
	}
	done := make(chan struct{})
	r.inFlight[k] = done
	var once sync.Once
	return func() {
		once.Do(func() {
			r.mutex.Lock()
			if cur, ok := r.inFlight[k]; ok && cur == done {
				delete(r.inFlight, k)
			}
			r.mutex.Unlock()
			close(done)
		})
	}
}

// Armed reports whether a create is in flight for this character.
func (r *CreateLatch) Armed(t tenant.Model, characterId character.Id) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	_, ok := r.inFlight[latchKey{tenant: t, characterId: characterId}]
	return ok
}

// AwaitSettled blocks until no create is in flight for this character, and
// reports whether it ever saw one.
//
// A create that has not armed YET is the case the goroutine-per-packet
// dispatch makes real: the invite's goroutine can reach this call before the
// create's goroutine has reached Begin, so an "is it armed right now?" test is
// not sufficient. It therefore waits up to `arrive` for one to appear before
// concluding there is none, and then up to `settle` for it to finish.
func (r *CreateLatch) AwaitSettled(t tenant.Model, characterId character.Id, arrive time.Duration, settle time.Duration) bool {
	done, ok := r.await(t, characterId, arrive)
	if !ok {
		return false
	}
	select {
	case <-done:
	case <-time.After(settle):
	}
	return true
}

// await polls for the latch to arm, returning its channel. Polling rather than
// signalling is deliberate: the waiter is looking for an event that has not
// happened yet on a key that does not exist yet, and the window it polls is
// bounded by a couple of hundred milliseconds once per trade open.
func (r *CreateLatch) await(t tenant.Model, characterId character.Id, arrive time.Duration) (chan struct{}, bool) {
	k := latchKey{tenant: t, characterId: characterId}
	deadline := time.Now().Add(arrive)
	for {
		r.mutex.Lock()
		done, ok := r.inFlight[k]
		r.mutex.Unlock()
		if ok {
			return done, true
		}
		if !time.Now().Before(deadline) {
			return nil, false
		}
		time.Sleep(latchPollInterval)
	}
}

const latchPollInterval = 2 * time.Millisecond
