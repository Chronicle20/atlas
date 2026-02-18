package invite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas-constants/world"
	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

var ErrNotFound = errors.New("not found")

const tenantTrackerKey = "invite:active-tenants"

type Registry struct {
	invites         *atlas.TenantRegistry[uint32, Model]
	idGen           *atlas.IDGenerator
	targetTypeIndex *atlas.Index       // "{targetId}:{inviteType}" → inviteId strings
	targetIndex     *atlas.Uint32Index // targetId → inviteIds
	originatorIndex *atlas.Uint32Index // originatorId → inviteIds
	client          *goredis.Client
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		invites: atlas.NewTenantRegistry[uint32, Model](client, "invite", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
		idGen:           atlas.NewIDGenerator(client, "invite"),
		targetTypeIndex: atlas.NewIndex(client, "invite", "target-type"),
		targetIndex:     atlas.NewUint32Index(client, "invite", "target"),
		originatorIndex: atlas.NewUint32Index(client, "invite", "originator"),
		client:          client,
	}
}

func GetRegistry() *Registry {
	return registry
}

func (r *Registry) trackTenant(ctx context.Context, t tenant.Model) {
	data, err := json.Marshal(&t)
	if err != nil {
		return
	}
	r.client.SAdd(ctx, tenantTrackerKey, string(data))
}

func (r *Registry) GetActiveTenants() []tenant.Model {
	ctx := context.Background()
	members, err := r.client.SMembers(ctx, tenantTrackerKey).Result()
	if err != nil {
		return nil
	}
	var tenants []tenant.Model
	for _, m := range members {
		var t tenant.Model
		if err := json.Unmarshal([]byte(m), &t); err != nil {
			continue
		}
		tenants = append(tenants, t)
	}
	return tenants
}

func targetTypeKey(targetId uint32, inviteType string) string {
	return fmt.Sprintf("%d:%s", targetId, inviteType)
}

func inviteIdStr(id uint32) string {
	return strconv.FormatUint(uint64(id), 10)
}

func parseInviteId(s string) (uint32, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(v), nil
}

func (r *Registry) Create(ctx context.Context, originatorId uint32, worldId world.Id, targetId uint32, inviteType string, referenceId uint32) Model {
	t := tenant.MustFromContext(ctx)

	// Check dedup by referenceId within the (target, type) bucket
	ttKey := targetTypeKey(targetId, inviteType)
	ids, _ := r.targetTypeIndex.Lookup(ctx, t, ttKey)
	for _, idStr := range ids {
		invId, err := parseInviteId(idStr)
		if err != nil {
			continue
		}
		existing, err := r.invites.Get(ctx, t, invId)
		if err != nil {
			continue
		}
		if existing.ReferenceId() == referenceId {
			return existing
		}
	}

	inviteId, _ := r.idGen.NextID(ctx, t)

	m, err := NewBuilder().
		SetTenant(t).
		SetId(inviteId).
		SetInviteType(inviteType).
		SetReferenceId(referenceId).
		SetOriginatorId(originatorId).
		SetTargetId(targetId).
		SetWorldId(worldId).
		SetAge(time.Now()).
		Build()
	if err != nil {
		panic("invite.Registry.Create: builder validation failed: " + err.Error())
	}

	_ = r.invites.Put(ctx, t, inviteId, m)
	_ = r.targetTypeIndex.Add(ctx, t, ttKey, inviteIdStr(inviteId))
	_ = r.targetIndex.Add(ctx, t, targetId, inviteId)
	_ = r.originatorIndex.Add(ctx, t, originatorId, inviteId)
	r.trackTenant(ctx, t)

	return m
}

func (r *Registry) GetByOriginator(ctx context.Context, actorId uint32, inviteType string, originatorId uint32) (Model, error) {
	t := tenant.MustFromContext(ctx)

	ttKey := targetTypeKey(actorId, inviteType)
	ids, _ := r.targetTypeIndex.Lookup(ctx, t, ttKey)
	for _, idStr := range ids {
		invId, err := parseInviteId(idStr)
		if err != nil {
			continue
		}
		m, err := r.invites.Get(ctx, t, invId)
		if err != nil {
			continue
		}
		if m.OriginatorId() == originatorId {
			return m, nil
		}
	}
	return Model{}, ErrNotFound
}

func (r *Registry) GetByReference(ctx context.Context, actorId uint32, inviteType string, referenceId uint32) (Model, error) {
	t := tenant.MustFromContext(ctx)

	ttKey := targetTypeKey(actorId, inviteType)
	ids, _ := r.targetTypeIndex.Lookup(ctx, t, ttKey)
	for _, idStr := range ids {
		invId, err := parseInviteId(idStr)
		if err != nil {
			continue
		}
		m, err := r.invites.Get(ctx, t, invId)
		if err != nil {
			continue
		}
		if m.ReferenceId() == referenceId {
			return m, nil
		}
	}
	return Model{}, ErrNotFound
}

func (r *Registry) GetForCharacter(ctx context.Context, characterId uint32) ([]Model, error) {
	t := tenant.MustFromContext(ctx)

	inviteIds, _ := r.targetIndex.Lookup(ctx, t, characterId)
	results := make([]Model, 0)
	for _, invId := range inviteIds {
		m, err := r.invites.Get(ctx, t, invId)
		if err != nil {
			continue
		}
		results = append(results, m)
	}
	return results, nil
}

func (r *Registry) Delete(ctx context.Context, actorId uint32, inviteType string, originatorId uint32) error {
	t := tenant.MustFromContext(ctx)

	ttKey := targetTypeKey(actorId, inviteType)
	ids, _ := r.targetTypeIndex.Lookup(ctx, t, ttKey)
	for _, idStr := range ids {
		invId, err := parseInviteId(idStr)
		if err != nil {
			continue
		}
		m, err := r.invites.Get(ctx, t, invId)
		if err != nil {
			continue
		}
		if m.OriginatorId() == originatorId {
			r.removeInvite(ctx, t, m)
			return nil
		}
	}
	return ErrNotFound
}

func (r *Registry) DeleteForCharacter(ctx context.Context, characterId uint32) []Model {
	t := tenant.MustFromContext(ctx)

	removed := make([]Model, 0)
	seen := make(map[uint32]bool)

	// Remove all invites targeting this character
	targetInviteIds, _ := r.targetIndex.Lookup(ctx, t, characterId)
	for _, invId := range targetInviteIds {
		if seen[invId] {
			continue
		}
		seen[invId] = true
		m, err := r.invites.Get(ctx, t, invId)
		if err != nil {
			continue
		}
		removed = append(removed, m)
		r.removeInvite(ctx, t, m)
	}

	// Remove all invites originated by this character
	originatorInviteIds, _ := r.originatorIndex.Lookup(ctx, t, characterId)
	for _, invId := range originatorInviteIds {
		if seen[invId] {
			continue
		}
		seen[invId] = true
		m, err := r.invites.Get(ctx, t, invId)
		if err != nil {
			continue
		}
		removed = append(removed, m)
		r.removeInvite(ctx, t, m)
	}

	return removed
}

func (r *Registry) GetExpired(ctx context.Context, timeout time.Duration) []Model {
	t := tenant.MustFromContext(ctx)

	all, err := r.invites.GetAllValues(ctx, t)
	if err != nil {
		return nil
	}

	results := make([]Model, 0)
	for _, m := range all {
		if m.Expired(timeout) {
			results = append(results, m)
		}
	}
	return results
}

func (r *Registry) removeInvite(ctx context.Context, t tenant.Model, m Model) {
	_ = r.invites.Remove(ctx, t, m.Id())
	_ = r.targetTypeIndex.Remove(ctx, t, targetTypeKey(m.TargetId(), m.Type()), inviteIdStr(m.Id()))
	_ = r.targetIndex.Remove(ctx, t, m.TargetId(), m.Id())
	_ = r.originatorIndex.Remove(ctx, t, m.OriginatorId(), m.Id())
}
