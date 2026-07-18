package berserk

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

var ErrNotFound = errors.New("not found")

// Registry stores tracked Dark Knights in Redis (namespace buffs-berserk) so
// state is shared across the service's replicas (design D1). Tenants are
// registered in the same buffs:_tenants set the buff registry maintains, so
// ticker fan-out sees tenants whose only tracked state is a Dark Knight.
type Registry struct {
	entries *atlas.TenantRegistry[uint32, Model]
	tenants *atlas.Set
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		entries: atlas.NewTenantRegistry[uint32, Model](client, "buffs-berserk", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
		tenants: atlas.NewSet(client, "buffs:_tenants"),
	}
}

func GetRegistry() *Registry {
	return registry
}

func (r *Registry) Track(ctx context.Context, m Model) error {
	t := tenant.MustFromContext(ctx)
	if err := r.entries.Put(ctx, t, m.CharacterId(), m); err != nil {
		return err
	}
	if tb, err := json.Marshal(&t); err == nil {
		_ = r.tenants.Add(ctx, string(tb))
	}
	return nil
}

func (r *Registry) Untrack(ctx context.Context, characterId uint32) error {
	t := tenant.MustFromContext(ctx)
	return r.entries.Remove(ctx, t, characterId)
}

func (r *Registry) Get(ctx context.Context, characterId uint32) (Model, error) {
	t := tenant.MustFromContext(ctx)
	m, err := r.entries.Get(ctx, t, characterId)
	if errors.Is(err, atlas.ErrNotFound) {
		return Model{}, ErrNotFound
	}
	return m, err
}

func (r *Registry) GetAll(ctx context.Context) []Model {
	t := tenant.MustFromContext(ctx)
	vals, err := r.entries.GetAllValues(ctx, t)
	if err != nil {
		return nil
	}
	return vals
}

func (r *Registry) GetTenants(ctx context.Context) ([]tenant.Model, error) {
	members, err := r.tenants.Members(ctx)
	if err != nil {
		return nil, err
	}
	var tenants []tenant.Model
	for _, mb := range members {
		var t tenant.Model
		if err := json.Unmarshal([]byte(mb), &t); err != nil {
			continue
		}
		tenants = append(tenants, t)
	}
	return tenants, nil
}

// maxUpdateAttempts bounds updateWithRetry: a lost WATCH race resolves on
// the very next read (nothing to wait out), so a small fixed cap is enough
// to absorb concurrent-writer contention without risking an unbounded loop.
const maxUpdateAttempts = 3

// updateWithRetry wraps r.entries.Update with a bounded retry on a lost
// WATCH race. TenantRegistry.Update (tenant_registry.go:130) is a single-
// attempt WATCH/GET/MULTI/EXEC: if another writer commits between our GET
// and EXEC, go-redis aborts the transaction and returns goredis.TxFailedErr,
// and the caller's write is silently dropped unless someone retries. For the
// mutator methods that route through this helper, a dropped write is a real
// defect (a lost mark/update/store), unlike the CLAIM methods where losing
// the race is the intended single-winner outcome (see ClaimReeval/
// ClaimBroadcast, which call r.entries.Update directly and must not retry).
//
// atlas.ErrNotFound is terminal (the entry isn't tracked) and is never
// retried. Any other error (connection failure, marshal error, etc.) is
// returned as-is after the first failure. fn is re-invoked on each attempt,
// which is safe because it recomputes its result from the freshly read
// value on every call.
func (r *Registry) updateWithRetry(ctx context.Context, t tenant.Model, characterId uint32, fn func(Model) Model) (Model, error) {
	var m Model
	var err error
	for attempt := 0; attempt < maxUpdateAttempts; attempt++ {
		m, err = r.entries.Update(ctx, t, characterId, fn)
		if err == nil || !errors.Is(err, goredis.TxFailedErr) {
			return m, err
		}
	}
	return m, err
}

// MarkDirty schedules a re-evaluation at/after `at`. Untracked characters are
// ignored (most characters are not Dark Knights). Last-writer-wins on dirtyAt
// among committed writes is intentional: re-evaluations are idempotent and
// compute from current data, so which trigger commits first is immaterial
// (design §5). A lost WATCH race is a different failure mode than that —
// nothing commits at all — so it is not a "loss" in the last-writer-wins
// sense; updateWithRetry retries it so the mark is not silently dropped.
func (r *Registry) MarkDirty(ctx context.Context, characterId uint32, at time.Time) error {
	t := tenant.MustFromContext(ctx)
	_, err := r.updateWithRetry(ctx, t, characterId, func(m Model) Model {
		return m.dirtyMarked(at)
	})
	if errors.Is(err, atlas.ErrNotFound) {
		return nil
	}
	return err
}

func (r *Registry) UpdateChannel(ctx context.Context, characterId uint32, worldId world.Id, channelId channel.Id) error {
	t := tenant.MustFromContext(ctx)
	_, err := r.updateWithRetry(ctx, t, characterId, func(m Model) Model {
		return m.channelUpdated(worldId, channelId)
	})
	if errors.Is(err, atlas.ErrNotFound) {
		return nil
	}
	return err
}

func (r *Registry) UpdateSkillLevel(ctx context.Context, characterId uint32, level byte) error {
	t := tenant.MustFromContext(ctx)
	_, err := r.updateWithRetry(ctx, t, characterId, func(m Model) Model {
		return m.skillLevelUpdated(level)
	})
	if errors.Is(err, atlas.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

// ClaimReeval atomically claims a due re-evaluation: it clears dirtyAt and
// returns (entry, true) iff the entry was dirty, due, and routable. Update is
// a single-attempt WATCH/MULTI (tenant_registry.go:130): when two replicas
// race, the loser's transaction fails and we report not-claimed — at most one
// re-evaluation runs per deadline (design D2).
func (r *Registry) ClaimReeval(ctx context.Context, characterId uint32, now time.Time) (Model, bool) {
	t := tenant.MustFromContext(ctx)
	claimed := false
	m, err := r.entries.Update(ctx, t, characterId, func(m Model) Model {
		claimed = false
		if m.DirtyDue(now) {
			claimed = true
			return m.dirtyCleared()
		}
		return m
	})
	if err != nil || !claimed {
		return Model{}, false
	}
	return m, true
}

// ClaimBroadcast atomically claims a due broadcast tick, advancing the
// deadline by BroadcastPeriod. Returns the claimed state to emit. Same
// single-winner semantics as ClaimReeval.
func (r *Registry) ClaimBroadcast(ctx context.Context, characterId uint32, now time.Time) (Model, bool) {
	t := tenant.MustFromContext(ctx)
	claimed := false
	m, err := r.entries.Update(ctx, t, characterId, func(m Model) Model {
		claimed = false
		if m.BroadcastDue(now) {
			claimed = true
			return m.broadcastScheduled(now.Add(BroadcastPeriod))
		}
		return m
	})
	if err != nil || !claimed {
		return Model{}, false
	}
	return m, true
}

// StoreEvaluation writes the outcome of a re-evaluation: the captured active
// state and refreshed character level. The broadcast schedule is only
// (re)started (at `now`) on a state transition or the first evaluation
// (Model.evaluated); an unchanged state preserves the running cadence so
// frequent HP re-evaluations don't starve the periodic re-broadcast
// (task-154 aura-starvation fix, revising design D2's cancel-and-replace).
func (r *Registry) StoreEvaluation(ctx context.Context, characterId uint32, active bool, characterLevel byte, now time.Time) error {
	t := tenant.MustFromContext(ctx)
	_, err := r.updateWithRetry(ctx, t, characterId, func(m Model) Model {
		return m.evaluated(active, characterLevel, now)
	})
	if errors.Is(err, atlas.ErrNotFound) {
		return nil
	}
	return err
}
