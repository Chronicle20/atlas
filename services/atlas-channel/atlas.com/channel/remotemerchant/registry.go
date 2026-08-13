// Package remotemerchant tracks which characters opened an NPC shop by using a
// classification-545 cash item (Miu Miu the Traveling Merchant) rather than by
// talking to the NPC.
//
// Why it exists: the client sets m_bExclRequestSent when it sends CASH_ITEM_USE
// and CShopDlg::SetShopDlg (@0x7529ad on v83) never clears it — decompiled in
// full during task-221's design phase (§1.2, OQ-2). So the server must send
// EnableActions. But the ordinary "talk to an NPC" shop path must stay
// byte-identical on the versions whose OPEN_NPC_SHOP matrix cells are already
// verified, so the unlock cannot be unconditional in the shop consumer. An
// entry here is the condition.
//
// This is presentation state only. Losing it (pod restart, dropped event) costs
// one EnableActions, never an item — the item's fate belongs entirely to the
// saga.
//
// Why in-process state is the whole view rather than a shard: a character's
// socket session lives on exactly one atlas-channel pod, so the pod that wrote
// the entry is the pod that owns the session that needs unlocking.
package remotemerchant

import (
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TTL bounds how long a pending unlock waits for its status event. A dropped or
// lost event would otherwise leave the character locked until they reconnect;
// the sweep unlocks them instead. 30s is well above the ENTER→ENTERED round
// trip and well below any tolerable stuck-client window.
const TTL = 30 * time.Second

type Key struct {
	Tenant      tenant.Model
	CharacterId uint32
}

// Entry is the pending unlock for one remote-initiated shop open.
type Entry struct {
	ItemId item.Id
	Slot   slot.Position
	At     time.Time
}

// Expired is one swept entry, carrying enough context to unlock its session.
type Expired struct {
	Tenant      tenant.Model
	CharacterId uint32
	Entry       Entry
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

// Put records a pending unlock. Called before the saga is created so a very
// fast ENTERED cannot arrive before the registry write.
func (r *Registry) Put(t tenant.Model, characterId uint32, e Entry) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.pending[Key{Tenant: t, CharacterId: characterId}] = e
}

// Take returns and removes the pending unlock in one lock, so an ENTERED and an
// ENTER_ERROR racing on the same character unlock exactly once.
func (r *Registry) Take(t tenant.Model, characterId uint32) (Entry, bool) {
	k := Key{Tenant: t, CharacterId: characterId}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	e, ok := r.pending[k]
	if ok {
		delete(r.pending, k)
	}
	return e, ok
}

// ClearCharacter drops the pending unlock for a character (session destroy).
// Without this the map leaks one entry per character ever seen by this pod.
func (r *Registry) ClearCharacter(t tenant.Model, characterId uint32) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.pending, Key{Tenant: t, CharacterId: characterId})
}

// Sweep removes and returns every entry older than TTL belonging to t. The
// caller sends EnableActions for each so a lost status event cannot leave a
// character permanently locked.
//
// Sweep is scoped to a single tenant deliberately: atlas-channel starts one
// sweep goroutine per (tenant, world, channel) listener key, and on a pod
// serving more than one tenant every one of those goroutines shares this same
// Registry. An unscoped Sweep is destructive — the first goroutine to fire
// would remove and claim every tenant's expired entries, and every other
// tenant's sweeper would see nothing to unlock for entries that were, in
// truth, never delivered to anyone (task-221 code review, round 2). Scoping
// the removal to t inside the lock means a tenant's entries can only ever be
// consumed by that tenant's own sweep.
func (r *Registry) Sweep(t tenant.Model, now time.Time) []Expired {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	var out []Expired
	for k, e := range r.pending {
		if !k.Tenant.Is(t) {
			continue
		}
		if now.Sub(e.At) >= TTL {
			out = append(out, Expired{Tenant: k.Tenant, CharacterId: k.CharacterId, Entry: e})
			delete(r.pending, k)
		}
	}
	return out
}
