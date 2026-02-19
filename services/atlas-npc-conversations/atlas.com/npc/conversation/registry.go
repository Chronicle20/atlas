package conversation

import (
	"context"
	"errors"
	"strconv"

	atlas "github.com/Chronicle20/atlas-redis"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	conversations *atlas.TenantRegistry[uint32, ConversationContext]
	sagaIndex     *atlas.TenantRegistry[uuid.UUID, uint32]
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		conversations: atlas.NewTenantRegistry[uint32, ConversationContext](client, "npc-conversation", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
		sagaIndex: atlas.NewTenantRegistry[uuid.UUID, uint32](client, "npc-conversation-saga", func(k uuid.UUID) string {
			return k.String()
		}),
	}
}

func GetRegistry() *Registry {
	return registry
}

func (s *Registry) GetPreviousContext(ctx context.Context, characterId uint32) (ConversationContext, error) {
	t := tenant.MustFromContext(ctx)
	val, err := s.conversations.Get(ctx, t, characterId)
	if err != nil {
		return ConversationContext{}, errors.New("unable to previous context")
	}
	return val, nil
}

func (s *Registry) SetContext(ctx context.Context, characterId uint32, cc ConversationContext) {
	t := tenant.MustFromContext(ctx)
	_ = s.conversations.Put(ctx, t, characterId, cc)
	if cc.PendingSagaId() != nil {
		_ = s.sagaIndex.Put(ctx, t, *cc.PendingSagaId(), characterId)
	}
}

func (s *Registry) ClearContext(ctx context.Context, characterId uint32) {
	t := tenant.MustFromContext(ctx)
	old, err := s.conversations.Get(ctx, t, characterId)
	if err == nil && old.PendingSagaId() != nil {
		_ = s.sagaIndex.Remove(ctx, t, *old.PendingSagaId())
	}
	_ = s.conversations.Remove(ctx, t, characterId)
}

func (s *Registry) UpdateContext(ctx context.Context, characterId uint32, cc ConversationContext) {
	t := tenant.MustFromContext(ctx)
	old, err := s.conversations.Get(ctx, t, characterId)
	if err == nil && old.PendingSagaId() != nil {
		if cc.PendingSagaId() == nil || *cc.PendingSagaId() != *old.PendingSagaId() {
			_ = s.sagaIndex.Remove(ctx, t, *old.PendingSagaId())
		}
	}
	_ = s.conversations.Put(ctx, t, characterId, cc)
	if cc.PendingSagaId() != nil {
		_ = s.sagaIndex.Put(ctx, t, *cc.PendingSagaId(), characterId)
	}
}

func (s *Registry) GetContextBySagaId(ctx context.Context, sagaId uuid.UUID) (ConversationContext, error) {
	t := tenant.MustFromContext(ctx)
	characterId, err := s.sagaIndex.Get(ctx, t, sagaId)
	if err != nil {
		return ConversationContext{}, errors.New("no conversation found for saga ID")
	}
	cc, err := s.conversations.Get(ctx, t, characterId)
	if err != nil {
		_ = s.sagaIndex.Remove(ctx, t, sagaId)
		return ConversationContext{}, errors.New("conversation context not found for character")
	}
	return cc, nil
}
