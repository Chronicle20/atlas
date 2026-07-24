// Package controller owns the single-controller-per-NPC election state
// (task-176, FR-5). Exactly one non-hidden character in a field is granted
// client-side control of each NPC; everyone else renders it as remote.
// State lives in Redis so every channel pod observes the same assignment
// (FR-5.5). Uncontrolled = absent from the hash (design D2) — there is no
// live-NPC record; static map data remains the NPC source of truth.
package controller

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Registry maps, per (tenant, field), NPC objectId -> controller
// characterId. Backing key:
// atlas:npc-controller:<tenantKey>:<world>:<channel>:<map>:<instance>.
type Registry struct {
	hash *atlasredis.TenantKeyedHash[string]
}

var (
	registry *Registry
	once     sync.Once
)

func newRegistry(rc *goredis.Client) *Registry {
	return &Registry{
		hash: atlasredis.NewTenantKeyedHash[string](rc, "npc-controller", func(s string) string { return s }),
	}
}

// InitRegistry initializes the singleton registry. Safe to call multiple
// times; only the first call takes effect (sync.Once).
func InitRegistry(rc *goredis.Client) {
	once.Do(func() {
		registry = newRegistry(rc)
	})
}

// GetRegistry returns the singleton, or nil before InitRegistry — callers
// must nil-check and fail open.
func GetRegistry() *Registry {
	return registry
}

func fieldSuffix(f field.Model) string {
	return fmt.Sprintf("%d:%d:%d:%s", byte(f.WorldId()), byte(f.ChannelId()), uint32(f.MapId()), f.Instance().String())
}

// Claim atomically records characterId as npcObjectId's controller iff no
// controller is recorded (HSETNX). Returns true when this call won.
func (r *Registry) Claim(ctx context.Context, t tenant.Model, f field.Model, npcObjectId uint32, characterId uint32) (bool, error) {
	return r.hash.SetNX(ctx, t, fieldSuffix(f), strconv.FormatUint(uint64(npcObjectId), 10), strconv.FormatUint(uint64(characterId), 10))
}

// Release removes the controller entries for the given NPCs. Idempotent;
// Redis deletes the hash when its last field goes, so empty fields leak
// nothing (no teardown sweep needed).
func (r *Registry) Release(ctx context.Context, t tenant.Model, f field.Model, npcObjectIds ...uint32) error {
	if len(npcObjectIds) == 0 {
		return nil
	}
	fields := make([]string, 0, len(npcObjectIds))
	for _, id := range npcObjectIds {
		fields = append(fields, strconv.FormatUint(uint64(id), 10))
	}
	return r.hash.Del(ctx, t, fieldSuffix(f), fields...)
}

// ControllerOf returns (controllerId, true) when npcObjectId has a recorded
// controller, (0, false) when uncontrolled.
func (r *Registry) ControllerOf(ctx context.Context, t tenant.Model, f field.Model, npcObjectId uint32) (uint32, bool, error) {
	v, err := r.hash.Get(ctx, t, fieldSuffix(f), strconv.FormatUint(uint64(npcObjectId), 10))
	if err != nil {
		if errors.Is(err, atlasredis.ErrNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}
	id, perr := strconv.ParseUint(v, 10, 32)
	if perr != nil {
		return 0, false, perr
	}
	return uint32(id), true, nil
}

// GetAll returns the full npcObjectId -> controllerId map for the field.
func (r *Registry) GetAll(ctx context.Context, t tenant.Model, f field.Model) (map[uint32]uint32, error) {
	raw, err := r.hash.GetAll(ctx, t, fieldSuffix(f))
	if err != nil {
		return nil, err
	}
	out := make(map[uint32]uint32, len(raw))
	for k, v := range raw {
		nid, e1 := strconv.ParseUint(k, 10, 32)
		cid, e2 := strconv.ParseUint(v, 10, 32)
		if e1 != nil || e2 != nil {
			continue
		}
		out[uint32(nid)] = uint32(cid)
	}
	return out, nil
}

// ControlledBy lists the NPCs currently assigned to characterId in field f.
func (r *Registry) ControlledBy(ctx context.Context, t tenant.Model, f field.Model, characterId uint32) ([]uint32, error) {
	all, err := r.GetAll(ctx, t, f)
	if err != nil {
		return nil, err
	}
	var out []uint32
	for nid, cid := range all {
		if cid == characterId {
			out = append(out, nid)
		}
	}
	return out, nil
}
