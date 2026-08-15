package coordinator

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Registry struct {
	active     *atlas.TenantSet                        // agreement-id strings
	agreements *atlas.TenantRegistry[uuid.UUID, Model] // agreement-id -> Model
	charAgree  *atlas.TenantRegistry[uint32, string]   // characterId -> agreement-id string
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		active:     atlas.NewTenantSet(client, "coordinator:active"),
		agreements: atlas.NewTenantRegistry[uuid.UUID, Model](client, "coordinator:agreement", func(id uuid.UUID) string { return id.String() }),
		charAgree:  atlas.NewTenantRegistry[uint32, string](client, "coordinator:char", func(id uint32) string { return strconv.FormatUint(uint64(id), 10) }),
	}
}

func GetRegistry() *Registry {
	return registry
}

func (r *Registry) Initiate(ctx context.Context, ch channel.Model, name string, leaderId uint32, members []uint32) error {
	t := tenant.MustFromContext(ctx)

	for _, m := range members {
		val, err := r.charAgree.Get(ctx, t, m)
		if err == nil && val != "" && val != uuid.Nil.String() {
			return errors.New("already attempting guild creation")
		}
	}

	agreementId := uuid.New()
	rm := make(map[uint32]bool)
	rm[leaderId] = true

	mdl := Model{
		tenant:    t,
		channel:   ch,
		leaderId:  leaderId,
		name:      name,
		requests:  members,
		responses: rm,
		age:       time.Now(),
	}

	for _, memberId := range members {
		if err := r.charAgree.Put(ctx, t, memberId, agreementId.String()); err != nil {
			return fmt.Errorf("track member agreement: %w", err)
		}
	}
	if err := r.agreements.Put(ctx, t, agreementId, mdl); err != nil {
		return fmt.Errorf("store agreement: %w", err)
	}
	return r.active.Add(ctx, t, agreementId.String())
}

func (r *Registry) Respond(ctx context.Context, characterId uint32, agree bool) (Model, error) {
	t := tenant.MustFromContext(ctx)

	agreementIdStr, err := r.charAgree.Get(ctx, t, characterId)
	if err != nil {
		return Model{}, fmt.Errorf("character not in agreement: %w", err)
	}
	agreementId, err := uuid.Parse(agreementIdStr)
	if err != nil {
		return Model{}, fmt.Errorf("parse agreement id: %w", err)
	}
	g, err := r.agreements.Get(ctx, t, agreementId)
	if err != nil {
		return Model{}, fmt.Errorf("agreement not found: %w", err)
	}

	if agree {
		g = g.Agree(characterId)
		_ = r.agreements.Put(ctx, t, agreementId, g)
		return g, nil
	}

	// Disagreed — delete the agreement and clear character mappings.
	_ = r.agreements.Remove(ctx, t, agreementId)
	_ = r.active.Remove(ctx, t, agreementId.String())
	for _, m := range g.requests {
		_ = r.charAgree.Put(ctx, t, m, uuid.Nil.String())
	}
	return g, nil
}

// GetExpiredAcrossTenants sweeps every tenant's pending agreements for ones
// older than timeout. Explicitly cross-tenant: its caller (the expiration
// ticker, guild/task.go) runs on context.Background() with no tenant in
// context, so there is no per-tenant context to loop over. Active and
// agreement entries are always written and removed together (Initiate,
// Respond-disagree), so enumerating r.agreements directly across tenants
// yields exactly the same set r.active would index — and each Model already
// carries its own tenant (Model.tenant), which is what the caller recovers
// from afterward.
func (r *Registry) GetExpiredAcrossTenants(timeout time.Duration) ([]Model, error) {
	ctx := context.Background()
	all, err := r.agreements.GetAllAcrossTenants(ctx)
	if err != nil {
		return nil, fmt.Errorf("get agreements across tenants: %w", err)
	}
	now := time.Now()
	results := make([]Model, 0)
	for _, g := range all {
		if now.Sub(g.Age()) > timeout {
			results = append(results, g)
		}
	}
	return results, nil
}
