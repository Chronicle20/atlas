package factory

import (
	"atlas-character-factory/configuration/tenant/characters/template"
	"context"
	"fmt"

	atlas "github.com/Chronicle20/atlas-redis"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

// FollowUpSagaTemplate stores the template information needed to create a follow-up saga
type FollowUpSagaTemplate struct {
	TenantId                       uuid.UUID          `json:"tenantId"`
	Input                          RestModel          `json:"input"`
	Template                       template.RestModel `json:"template"`
	CharacterCreationTransactionId uuid.UUID          `json:"characterCreationTransactionId"`
}

// FollowUpSagaTemplateStore provides Redis-backed storage for follow-up saga templates
type FollowUpSagaTemplateStore struct {
	reg *atlas.Registry[string, FollowUpSagaTemplate]
}

var templateStoreInstance *FollowUpSagaTemplateStore

// GetFollowUpSagaTemplateStore returns the singleton instance of the template store
func GetFollowUpSagaTemplateStore() *FollowUpSagaTemplateStore {
	return templateStoreInstance
}

// Store stores the template information for later use when character created event is received
func (s *FollowUpSagaTemplateStore) Store(tenantId uuid.UUID, characterName string, input RestModel, tmpl template.RestModel, characterCreationTransactionId uuid.UUID) error {
	key := fmt.Sprintf("%s:%s", tenantId.String(), characterName)
	return s.reg.Put(context.Background(), key, FollowUpSagaTemplate{
		TenantId:                       tenantId,
		Input:                          input,
		Template:                       tmpl,
		CharacterCreationTransactionId: characterCreationTransactionId,
	})
}

// Get retrieves the stored template information
func (s *FollowUpSagaTemplateStore) Get(tenantId uuid.UUID, characterName string) (FollowUpSagaTemplate, bool) {
	key := fmt.Sprintf("%s:%s", tenantId.String(), characterName)
	v, err := s.reg.Get(context.Background(), key)
	if err != nil {
		return FollowUpSagaTemplate{}, false
	}
	return v, true
}

// Remove removes the stored template information after use
func (s *FollowUpSagaTemplateStore) Remove(tenantId uuid.UUID, characterName string) {
	key := fmt.Sprintf("%s:%s", tenantId.String(), characterName)
	_ = s.reg.Remove(context.Background(), key)
}

// Clear removes all stored templates
func (s *FollowUpSagaTemplateStore) Clear() {
	ctx := context.Background()
	pattern := fmt.Sprintf("atlas:%s:*", s.reg.Namespace())
	var cursor uint64
	for {
		keys, next, err := s.reg.Client().Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			break
		}
		if len(keys) > 0 {
			s.reg.Client().Del(ctx, keys...)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
}

// Size returns the number of stored templates
func (s *FollowUpSagaTemplateStore) Size() int {
	ctx := context.Background()
	pattern := fmt.Sprintf("atlas:%s:*", s.reg.Namespace())
	count := 0
	var cursor uint64
	for {
		keys, next, err := s.reg.Client().Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			break
		}
		count += len(keys)
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return count
}

// storeFollowUpSagaTemplate stores the template information for later use when character created event is received
func storeFollowUpSagaTemplate(ctx context.Context, characterName string, input RestModel, tmpl template.RestModel, characterCreationTransactionId uuid.UUID) error {
	t := tenant.MustFromContext(ctx)
	store := GetFollowUpSagaTemplateStore()
	return store.Store(t.Id(), characterName, input, tmpl, characterCreationTransactionId)
}

// GetFollowUpSagaTemplate retrieves the stored template information
func GetFollowUpSagaTemplate(tenantId uuid.UUID, characterName string) (FollowUpSagaTemplate, bool) {
	store := GetFollowUpSagaTemplateStore()
	return store.Get(tenantId, characterName)
}

// RemoveFollowUpSagaTemplate removes the stored template information after use
func RemoveFollowUpSagaTemplate(tenantId uuid.UUID, characterName string) {
	store := GetFollowUpSagaTemplateStore()
	store.Remove(tenantId, characterName)
}

// SagaCompletionTracker tracks completion status for character creation saga pairs
type SagaCompletionTracker struct {
	TenantId                       uuid.UUID `json:"tenantId"`
	AccountId                      uint32    `json:"accountId"`
	CharacterId                    uint32    `json:"characterId"`
	CharacterCreationTransactionId uuid.UUID `json:"characterCreationTransactionId"`
	FollowUpSagaTransactionId      uuid.UUID `json:"followUpSagaTransactionId"`
	CharacterCreationCompleted     bool      `json:"characterCreationCompleted"`
	FollowUpSagaCompleted          bool      `json:"followUpSagaCompleted"`
}

// SagaCompletionTrackerStore provides Redis-backed storage for saga completion tracking
type SagaCompletionTrackerStore struct {
	reg *atlas.Registry[string, SagaCompletionTracker]
}

var sagaTrackerStoreInstance *SagaCompletionTrackerStore

// GetSagaCompletionTrackerStore returns the singleton instance of the saga completion tracker store
func GetSagaCompletionTrackerStore() *SagaCompletionTrackerStore {
	return sagaTrackerStoreInstance
}

// StoreTrackerForCharacterCreation stores tracking information for character creation saga
func (s *SagaCompletionTrackerStore) StoreTrackerForCharacterCreation(tenantId uuid.UUID, accountId uint32, characterCreationTransactionId uuid.UUID) {
	ctx := context.Background()
	tracker := SagaCompletionTracker{
		TenantId:                       tenantId,
		AccountId:                      accountId,
		CharacterCreationTransactionId: characterCreationTransactionId,
	}
	_ = s.reg.Put(ctx, characterCreationTransactionId.String(), tracker)
}

// UpdateTrackerForFollowUpSaga updates the tracker with follow-up saga information
func (s *SagaCompletionTrackerStore) UpdateTrackerForFollowUpSaga(characterCreationTransactionId uuid.UUID, followUpSagaTransactionId uuid.UUID, characterId uint32) {
	ctx := context.Background()
	tracker, err := s.reg.Get(ctx, characterCreationTransactionId.String())
	if err != nil {
		return
	}
	tracker.FollowUpSagaTransactionId = followUpSagaTransactionId
	tracker.CharacterId = characterId
	_ = s.reg.Put(ctx, characterCreationTransactionId.String(), tracker)
	_ = s.reg.Put(ctx, followUpSagaTransactionId.String(), tracker)
}

// MarkSagaCompleted marks a saga as completed and returns the tracker if both sagas are now complete
func (s *SagaCompletionTrackerStore) MarkSagaCompleted(transactionId uuid.UUID) (*SagaCompletionTracker, bool) {
	ctx := context.Background()
	tracker, err := s.reg.Get(ctx, transactionId.String())
	if err != nil {
		return nil, false
	}

	if transactionId == tracker.CharacterCreationTransactionId {
		tracker.CharacterCreationCompleted = true
	} else if transactionId == tracker.FollowUpSagaTransactionId {
		tracker.FollowUpSagaCompleted = true
	}

	if tracker.CharacterCreationCompleted && tracker.FollowUpSagaCompleted {
		_ = s.reg.Remove(ctx, tracker.CharacterCreationTransactionId.String())
		if tracker.FollowUpSagaTransactionId != uuid.Nil {
			_ = s.reg.Remove(ctx, tracker.FollowUpSagaTransactionId.String())
		}
		return &tracker, true
	}

	// Write updated tracker back to both keys
	_ = s.reg.Put(ctx, tracker.CharacterCreationTransactionId.String(), tracker)
	if tracker.FollowUpSagaTransactionId != uuid.Nil {
		_ = s.reg.Put(ctx, tracker.FollowUpSagaTransactionId.String(), tracker)
	}

	return nil, false
}

// Clear removes all stored trackers
func (s *SagaCompletionTrackerStore) Clear() {
	ctx := context.Background()
	pattern := fmt.Sprintf("atlas:%s:*", s.reg.Namespace())
	var cursor uint64
	for {
		keys, next, err := s.reg.Client().Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			break
		}
		if len(keys) > 0 {
			s.reg.Client().Del(ctx, keys...)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
}

// Size returns the number of stored trackers
func (s *SagaCompletionTrackerStore) Size() int {
	ctx := context.Background()
	pattern := fmt.Sprintf("atlas:%s:*", s.reg.Namespace())
	count := 0
	var cursor uint64
	for {
		keys, next, err := s.reg.Client().Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			break
		}
		count += len(keys)
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return count
}

// InitCache initializes both Redis-backed stores
func InitCache(client *goredis.Client) {
	templateStoreInstance = &FollowUpSagaTemplateStore{
		reg: atlas.NewRegistry[string, FollowUpSagaTemplate](client, "char-factory-template", func(k string) string { return k }),
	}
	sagaTrackerStoreInstance = &SagaCompletionTrackerStore{
		reg: atlas.NewRegistry[string, SagaCompletionTracker](client, "char-factory-saga", func(k string) string { return k }),
	}
}

// StoreFollowUpSagaTracking stores the follow-up saga tracking information
func StoreFollowUpSagaTracking(characterCreationTransactionId uuid.UUID, followUpSagaTransactionId uuid.UUID, characterId uint32) {
	store := GetSagaCompletionTrackerStore()
	store.UpdateTrackerForFollowUpSaga(characterCreationTransactionId, followUpSagaTransactionId, characterId)
}

// MarkSagaCompleted marks a saga as completed and returns completion information
func MarkSagaCompleted(transactionId uuid.UUID) (*SagaCompletionTracker, bool) {
	store := GetSagaCompletionTrackerStore()
	return store.MarkSagaCompleted(transactionId)
}
