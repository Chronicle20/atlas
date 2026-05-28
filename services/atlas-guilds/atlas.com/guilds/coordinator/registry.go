package coordinator

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	active     *atlas.Set                            // agreement-id strings
	agreements *atlas.Registry[uuid.UUID, Model]     // agreement-id -> Model
	charAgree  *atlas.TenantRegistry[uint32, string] // characterId -> agreement-id string
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		active:     atlas.NewSet(client, "coordinator:active"),
		agreements: atlas.NewRegistry[uuid.UUID, Model](client, "coordinator:agreement", func(id uuid.UUID) string { return id.String() }),
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
	if err := r.agreements.Put(ctx, agreementId, mdl); err != nil {
		return fmt.Errorf("store agreement: %w", err)
	}
	return r.active.Add(ctx, agreementId.String())
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
	g, err := r.agreements.Get(ctx, agreementId)
	if err != nil {
		return Model{}, fmt.Errorf("agreement not found: %w", err)
	}

	if agree {
		g = g.Agree(characterId)
		_ = r.agreements.Put(ctx, agreementId, g)
		return g, nil
	}

	// Disagreed — delete the agreement and clear character mappings.
	_ = r.agreements.Remove(ctx, agreementId)
	_ = r.active.Remove(ctx, agreementId.String())
	for _, m := range g.requests {
		_ = r.charAgree.Put(ctx, t, m, uuid.Nil.String())
	}
	return g, nil
}

func (r *Registry) GetExpired(timeout time.Duration) ([]Model, error) {
	ctx := context.Background()
	members, err := r.active.Members(ctx)
	if err != nil {
		return nil, fmt.Errorf("get active agreements: %w", err)
	}
	now := time.Now()
	results := make([]Model, 0)
	for _, idStr := range members {
		id, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}
		g, err := r.agreements.Get(ctx, id)
		if err != nil {
			continue
		}
		if now.Sub(g.Age()) > timeout {
			results = append(results, g)
		}
	}
	return results, nil
}
