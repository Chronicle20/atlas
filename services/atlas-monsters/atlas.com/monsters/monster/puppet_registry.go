package monster

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// PuppetVicinityDistanceSq is the squared-distance threshold (Cosmic
// Monster.java isPuppetInVicinity, distanceSq < 177777) within which a player's
// puppet is considered to be covering a monster, biasing controller selection
// toward that puppet's owner.
const PuppetVicinityDistanceSq = 177777

// storedPuppet is the JSON-serializable representation of a puppet's owner and
// position, stored per (field, owner) in Redis.
type storedPuppet struct {
	OwnerCharacterId uint32 `json:"ownerCharacterId"`
	X                int16  `json:"x"`
	Y                int16  `json:"y"`
}

// PuppetRegistry tracks the live puppets per field so the monster controller
// picker can prefer a nearby puppet's owner. It mirrors the monster Registry's
// reg + mapIdx pairing: a value Registry keyed by (field, owner) for the
// {ownerCharacterId, x, y} payload, plus a per-field SET index of owner ids so
// puppets can be enumerated and removed by owner.
type PuppetRegistry struct {
	// reg backs the puppet payload store. namespace "monster-puppet", identity
	// keyFn, so the stored key is
	// atlas:monster-puppet:<tenantId>:<world>:<channel>:<map>:<instance>:<owner>.
	reg *atlasredis.Registry[string, storedPuppet]
	// fieldIdx backs the per-field membership SET of owner character ids.
	// namespace "monster-puppet-field", identity keyFn, so the SET key is
	// atlas:monster-puppet-field:<tenantId>:<world>:<channel>:<map>:<instance>.
	fieldIdx *atlasredis.KeyedSet[string]
}

var (
	puppetRegistry *PuppetRegistry
	puppetOnce     sync.Once
)

func InitPuppetRegistry(rc *goredis.Client) {
	puppetOnce.Do(func() {
		puppetRegistry = &PuppetRegistry{
			reg:      atlasredis.NewRegistry[string, storedPuppet](rc, "monster-puppet", func(s string) string { return s }),
			fieldIdx: atlasredis.NewKeyedSet[string](rc, "monster-puppet-field", func(s string) string { return s }),
		}
	})
}

func GetPuppetRegistry() *PuppetRegistry {
	return puppetRegistry
}

// puppetFieldSuffix reproduces the per-field SET key tail:
// <tenantId>:<world>:<channel>:<map>:<instance>.
func puppetFieldSuffix(t tenant.Model, f field.Model) string {
	return fmt.Sprintf("%s:%d:%d:%d:%s", t.Id().String(), byte(f.WorldId()), byte(f.ChannelId()), uint32(f.MapId()), f.Instance().String())
}

// puppetSuffix reproduces the per-(field, owner) payload key tail.
func puppetSuffix(t tenant.Model, f field.Model, ownerCharacterId uint32) string {
	return fmt.Sprintf("%s:%d", puppetFieldSuffix(t, f), ownerCharacterId)
}

// Add records (or replaces) the puppet for ownerCharacterId in field f at (x,y).
func (r *PuppetRegistry) Add(ctx context.Context, t tenant.Model, f field.Model, ownerCharacterId uint32, x int16, y int16) {
	_ = r.reg.Put(ctx, puppetSuffix(t, f, ownerCharacterId), storedPuppet{
		OwnerCharacterId: ownerCharacterId,
		X:                x,
		Y:                y,
	})
	_ = r.fieldIdx.Add(ctx, puppetFieldSuffix(t, f), strconv.FormatUint(uint64(ownerCharacterId), 10))
}

// Remove deletes the puppet for ownerCharacterId in field f.
func (r *PuppetRegistry) Remove(ctx context.Context, t tenant.Model, f field.Model, ownerCharacterId uint32) {
	_ = r.reg.Remove(ctx, puppetSuffix(t, f, ownerCharacterId))
	_ = r.fieldIdx.Remove(ctx, puppetFieldSuffix(t, f), strconv.FormatUint(uint64(ownerCharacterId), 10))
}

// GetInField returns every puppet currently registered in field f.
func (r *PuppetRegistry) GetInField(ctx context.Context, t tenant.Model, f field.Model) []storedPuppet {
	members, err := r.fieldIdx.Members(ctx, puppetFieldSuffix(t, f))
	if err != nil || len(members) == 0 {
		return nil
	}
	result := make([]storedPuppet, 0, len(members))
	for _, ownerStr := range members {
		ownerId, perr := strconv.ParseUint(ownerStr, 10, 32)
		if perr != nil {
			continue
		}
		sp, gerr := r.reg.Get(ctx, puppetSuffix(t, f, uint32(ownerId)))
		if gerr != nil {
			continue
		}
		result = append(result, sp)
	}
	return result
}

// VicinityOwner returns the owner character id of a puppet in field f that lies
// within PuppetVicinityDistanceSq of (x,y), and true; otherwise (0, false). When
// multiple puppets qualify the nearest is returned for determinism.
func (r *PuppetRegistry) VicinityOwner(ctx context.Context, t tenant.Model, f field.Model, x int16, y int16) (uint32, bool) {
	puppets := r.GetInField(ctx, t, f)
	var best uint32
	bestDistSq := int64(PuppetVicinityDistanceSq)
	found := false
	for _, sp := range puppets {
		dx := int64(sp.X) - int64(x)
		dy := int64(sp.Y) - int64(y)
		distSq := dx*dx + dy*dy
		if distSq < bestDistSq {
			bestDistSq = distSq
			best = sp.OwnerCharacterId
			found = true
		}
	}
	return best, found
}

// Clear removes all puppet state (payloads + field indexes).
func (r *PuppetRegistry) Clear(ctx context.Context) {
	_, _ = r.reg.Clear(ctx)
	_, _ = r.fieldIdx.ClearAll(ctx)
}
