package coordinator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const (
	agreementKeyPrefix = "coordinator:agreement:"
	activeSetKey       = "coordinator:active"
)

type Registry struct {
	client *goredis.Client
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{client: client}
}

func GetRegistry() *Registry {
	return registry
}

func charKey(tenantKey string, characterId uint32) string {
	return fmt.Sprintf("coordinator:char:%s:%d", tenantKey, characterId)
}

func agreementKey(agreementId uuid.UUID) string {
	return agreementKeyPrefix + agreementId.String()
}

func (r *Registry) Initiate(ctx context.Context, ch channel.Model, name string, leaderId uint32, members []uint32) error {
	t := tenant.MustFromContext(ctx)
	tk := atlas.TenantKey(t)

	// Check no member has an active agreement
	for _, m := range members {
		val, err := r.client.Get(ctx, charKey(tk, m)).Result()
		if err == nil && val != "" && val != uuid.Nil.String() {
			return errors.New("already attempting guild creation")
		}
	}

	agreementId := uuid.New()

	rm := make(map[uint32]bool)
	rm[leaderId] = true

	m := Model{
		tenant:    t,
		channel:   ch,
		leaderId:  leaderId,
		name:      name,
		requests:  members,
		responses: rm,
		age:       time.Now(),
	}

	data, err := json.Marshal(&m)
	if err != nil {
		return fmt.Errorf("marshal agreement: %w", err)
	}

	pipe := r.client.Pipeline()
	for _, memberId := range members {
		pipe.Set(ctx, charKey(tk, memberId), agreementId.String(), 0)
	}
	pipe.Set(ctx, agreementKey(agreementId), data, 0)
	pipe.SAdd(ctx, activeSetKey, agreementId.String())
	_, err = pipe.Exec(ctx)
	return err
}

func (r *Registry) Respond(ctx context.Context, characterId uint32, agree bool) (Model, error) {
	t := tenant.MustFromContext(ctx)
	tk := atlas.TenantKey(t)

	agreementIdStr, err := r.client.Get(ctx, charKey(tk, characterId)).Result()
	if err != nil {
		return Model{}, fmt.Errorf("character not in agreement: %w", err)
	}

	agreementId, err := uuid.Parse(agreementIdStr)
	if err != nil {
		return Model{}, fmt.Errorf("parse agreement id: %w", err)
	}

	data, err := r.client.Get(ctx, agreementKey(agreementId)).Bytes()
	if err != nil {
		return Model{}, fmt.Errorf("agreement not found: %w", err)
	}

	var g Model
	if err := json.Unmarshal(data, &g); err != nil {
		return Model{}, fmt.Errorf("unmarshal agreement: %w", err)
	}

	if agree {
		g = g.Agree(characterId)
		// Write back the updated agreement
		updatedData, err := json.Marshal(&g)
		if err != nil {
			return g, nil
		}
		_ = r.client.Set(ctx, agreementKey(agreementId), updatedData, 0).Err()
		return g, nil
	}

	// Disagreed — delete the agreement and clear character mappings
	pipe := r.client.Pipeline()
	pipe.Del(ctx, agreementKey(agreementId))
	pipe.SRem(ctx, activeSetKey, agreementId.String())
	for _, m := range g.requests {
		pipe.Set(ctx, charKey(tk, m), uuid.Nil.String(), 0)
	}
	_, _ = pipe.Exec(ctx)
	return g, nil
}

func (r *Registry) GetExpired(timeout time.Duration) ([]Model, error) {
	ctx := context.Background()

	members, err := r.client.SMembers(ctx, activeSetKey).Result()
	if err != nil {
		return nil, fmt.Errorf("get active agreements: %w", err)
	}

	now := time.Now()
	results := make([]Model, 0)

	for _, idStr := range members {
		data, err := r.client.Get(ctx, agreementKeyPrefix+idStr).Bytes()
		if err != nil {
			continue
		}
		var g Model
		if err := json.Unmarshal(data, &g); err != nil {
			continue
		}
		if now.Sub(g.Age()) > timeout {
			results = append(results, g)
		}
	}
	return results, nil
}
