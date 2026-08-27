// Package maplelife tracks the in-flight Maple Life character-creation dialog
// for an account whose player has not yet created a character. It is keyed by
// AccountId — not CharacterId — because the whole point of the dialog is that
// no character exists yet.
//
// Why in-process state is the whole view rather than a shard: an account's
// socket session lives on exactly one atlas-channel pod, so the pod that wrote
// the entry is the pod that owns the session driving the dialog.
//
// This is presentation state only. Losing it (pod restart, dropped event)
// costs one dialog, never an item — the item's fate belongs to the saga and
// to the pre-check ordering, not to this registry.
package maplelife

import (
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Phase is where a pending dialog sits in its lifecycle.
type Phase string

const (
	// PhaseSubmitted is a dialog whose creation request has been sent to the
	// character-creation saga and is awaiting CREATED/FAILED. It is the ONLY
	// phase an entry is ever created in: there is no open-time packet at all
	// (bug-543-is-the-submit-not-the-open.md) -- the client opens its own
	// CUICharacterSaleDlg locally, and the first serverbound signal this
	// registry ever sees for an account is already the submit.
	PhaseSubmitted Phase = "SUBMITTED"
)

// SubmittedTTL bounds how long a submitted dialog waits for the saga's
// CREATED/FAILED outcome. It must outlive the orchestrator's 10s
// character-creation backstop (design §4.2): if the saga times out and emits
// FAILED after this registry has already swept the entry, the FAILED handler
// has nothing left to correlate against. 30s gives comfortable headroom above
// that 10s backstop.
const SubmittedTTL = 30 * time.Second

type Key struct {
	Tenant    tenant.Model
	AccountId uint32
}

// Entry is the pending Maple Life dialog for one account.
type Entry struct {
	CharacterId   uint32
	WorldId       world.Id
	ItemId        item.Id
	Slot          slot.Position
	UpdateTime    uint32
	Phase         Phase
	TransactionId string
	CandidateName string
	At            time.Time
}

// Expired is one swept entry, carrying enough context to notify its session.
type Expired struct {
	Tenant    tenant.Model
	AccountId uint32
	Entry     Entry
}

type Registry struct {
	mutex   sync.RWMutex
	pending map[Key]Entry
}

var (
	registry *Registry
	once     sync.Once
)

func GetRegistry() *Registry {
	once.Do(func() {
		registry = &Registry{pending: make(map[Key]Entry)}
	})
	return registry
}

// Put records a pending dialog for an account, replacing any existing one.
// A second Open for the same account refreshes rather than duplicates
// (design §3), so the second Put's values always win.
func (r *Registry) Put(t tenant.Model, accountId uint32, e Entry) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.pending[Key{Tenant: t, AccountId: accountId}] = e
}

// Get returns the pending dialog for an account without removing it.
func (r *Registry) Get(t tenant.Model, accountId uint32) (Entry, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	e, ok := r.pending[Key{Tenant: t, AccountId: accountId}]
	return e, ok
}

// Take returns and removes the pending dialog in one lock, so a CREATED and a
// FAILED racing on the same account consume exactly once.
func (r *Registry) Take(t tenant.Model, accountId uint32) (Entry, bool) {
	k := Key{Tenant: t, AccountId: accountId}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	e, ok := r.pending[k]
	if ok {
		delete(r.pending, k)
	}
	return e, ok
}

// TakeByTransactionId returns and removes the pending dialog matching
// transactionId for tenant t, along with the account it belongs to. An empty
// transactionId never matches, even against an entry whose TransactionId is
// also empty, since an empty id identifies no saga.
func (r *Registry) TakeByTransactionId(t tenant.Model, transactionId string) (uint32, Entry, bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if transactionId == "" {
		return 0, Entry{}, false
	}
	for k, e := range r.pending {
		if !k.Tenant.Is(t) {
			continue
		}
		if e.TransactionId == transactionId {
			delete(r.pending, k)
			return k.AccountId, e, true
		}
	}
	return 0, Entry{}, false
}

// ClearAccount drops the pending dialog for an account (session destroy).
// Without this the map leaks one entry per account ever seen by this pod.
func (r *Registry) ClearAccount(t tenant.Model, accountId uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.pending, Key{Tenant: t, AccountId: accountId})
}

// Sweep removes and returns every entry belonging to t whose age has passed
// SubmittedTTL -- the only phase an entry now exists in. The caller notifies
// each expired account so a lost status event cannot leave a session
// permanently stuck.
//
// Sweep is scoped to a single tenant deliberately: atlas-channel starts one
// sweep goroutine per (tenant, world, channel) listener key, and on a pod
// serving more than one tenant every one of those goroutines shares this same
// Registry. An unscoped Sweep is destructive — the first goroutine to fire
// would remove and claim every tenant's expired entries, and every other
// tenant's sweeper would see nothing to act on for entries that were, in
// truth, never delivered to anyone. Scoping the removal to t inside the lock
// means a tenant's entries can only ever be consumed by that tenant's own
// sweep.
func (r *Registry) Sweep(t tenant.Model, now time.Time) []Expired {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	var out []Expired
	for k, e := range r.pending {
		if !k.Tenant.Is(t) {
			continue
		}
		if now.Sub(e.At) >= SubmittedTTL {
			out = append(out, Expired{Tenant: k.Tenant, AccountId: k.AccountId, Entry: e})
			delete(r.pending, k)
		}
	}
	return out
}
