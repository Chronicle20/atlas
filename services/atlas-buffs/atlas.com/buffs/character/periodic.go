package character

import (
	"atlas-buffs/buff/stat"
	"atlas-buffs/periodic"
	"context"
	"sort"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// PeriodicTickTTL bounds a leaked throttle entry. Every live row refreshes its
// key at most one interval (<= 5s) apart, so an active entry never lapses; an
// entry whose owning buff vanished by a removal path we failed to wire
// evaporates on its own. This is belt-and-braces for FR-6.2 — the explicit
// clears in ClearPeriodicTicksFor are still the mechanism, the TTL just makes a
// missed clear "stale for <= 5 min" instead of "leaked forever".
const PeriodicTickTTL = 5 * time.Minute

// TickKey identifies one periodic effect on one character. Keying by stat type
// as well as character is what lets two periodic effects on the same character
// throttle independently (FR-2.2); the pre-task-214 poison store was keyed by
// character alone.
type TickKey struct {
	CharacterId uint32
	StatType    string
}

// PeriodicEntry is one due-able periodic effect found by the scan.
type PeriodicEntry struct {
	Tenant      tenant.Model
	WorldId     world.Id
	ChannelId   channel.Id
	CharacterId uint32
	StatType    string
	Amount      int32
}

// GetPeriodicEntries does ONE traversal of the tenant's stored characters and
// yields every (character, statType) whose stat type has a periodic-effect row
// and whose owning buff has not expired (FR-2.1). Adding a table row adds no
// scan pass.
//
// When two live buffs carry the same periodic stat type for one character, the
// largest Amount wins. Buffs are stored in a Go map, so a first-wins rule would
// pick a different buff on different passes; max-wins is deterministic. With a
// single buff — every real case today — the result is identical to the
// pre-task-214 poison scan.
//
// Results are sorted by (CharacterId, StatType) so a tick pass emits in a
// stable order.
func (r *Registry) GetPeriodicEntries(ctx context.Context) []PeriodicEntry {
	t := tenant.MustFromContext(ctx)
	vals, err := r.characters.GetAllValues(ctx, t)
	if err != nil {
		return nil
	}

	best := make(map[TickKey]PeriodicEntry)
	for _, m := range vals {
		for _, b := range m.buffs {
			if b.Expired() {
				continue
			}
			for _, c := range b.Changes() {
				if _, ok := periodic.Lookup(c.Type()); !ok {
					continue
				}
				k := TickKey{CharacterId: m.characterId, StatType: c.Type()}
				if cur, seen := best[k]; seen && cur.Amount >= c.Amount() {
					continue
				}
				best[k] = PeriodicEntry{
					Tenant:      t,
					WorldId:     m.worldId,
					ChannelId:   m.channelId,
					CharacterId: m.characterId,
					StatType:    c.Type(),
					Amount:      c.Amount(),
				}
			}
		}
	}

	results := make([]PeriodicEntry, 0, len(best))
	for _, e := range best {
		results = append(results, e)
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].CharacterId != results[j].CharacterId {
			return results[i].CharacterId < results[j].CharacterId
		}
		return results[i].StatType < results[j].StatType
	})
	return results
}

// GetPeriodicTick reports when this effect last ticked for this character.
func (r *Registry) GetPeriodicTick(ctx context.Context, key TickKey) (time.Time, bool) {
	t := tenant.MustFromContext(ctx)
	at, err := r.periodicTicks.Get(ctx, t, key)
	if err != nil {
		return time.Time{}, false
	}
	return at, true
}

// UpdatePeriodicTick records a tick.
func (r *Registry) UpdatePeriodicTick(ctx context.Context, key TickKey, at time.Time) {
	t := tenant.MustFromContext(ctx)
	_ = r.periodicTicks.PutWithTTL(ctx, t, key, at, PeriodicTickTTL)
}

// ClearPeriodicTick drops one effect's throttle entry.
func (r *Registry) ClearPeriodicTick(ctx context.Context, key TickKey) {
	t := tenant.MustFromContext(ctx)
	_ = r.periodicTicks.Remove(ctx, t, key)
}

// ClearPeriodicTicksFor drops the throttle entry for every periodic stat type
// carried by the removed buffs' change sets (FR-6.1). Callers pass the
// Changes() of each cancelled/expired buff, mirroring the variadic shape
// markBerserkDirtyOnMaxHpChange already uses in the same removal paths.
func (r *Registry) ClearPeriodicTicksFor(ctx context.Context, characterId uint32, changeSets ...[]stat.Model) {
	for _, changes := range changeSets {
		for _, c := range changes {
			if _, ok := periodic.Lookup(c.Type()); !ok {
				continue
			}
			r.ClearPeriodicTick(ctx, TickKey{CharacterId: characterId, StatType: c.Type()})
		}
	}
}
