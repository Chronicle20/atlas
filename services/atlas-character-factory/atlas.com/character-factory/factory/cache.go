package factory

import (
	"atlas-character-factory/configuration/tenant/characters/template"
	"context"
	"fmt"
	"sync"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

// FollowUpSagaTemplate stores the template information needed to create a follow-up saga
type FollowUpSagaTemplate struct {
	TenantId                       uuid.UUID
	Input                          RestModel
	Template                       template.RestModel
	CharacterCreationTransactionId uuid.UUID
}

// FollowUpSagaTemplateStore provides thread-safe storage for follow-up saga templates
type FollowUpSagaTemplateStore struct {
	templates map[string]FollowUpSagaTemplate
	mutex     sync.RWMutex
}

// Singleton instance
var (
	templateStoreInstance *FollowUpSagaTemplateStore
	templateStoreOnce     sync.Once
)

// GetFollowUpSagaTemplateStore returns the singleton instance of the template store
func GetFollowUpSagaTemplateStore() *FollowUpSagaTemplateStore {
	templateStoreOnce.Do(func() {
		templateStoreInstance = &FollowUpSagaTemplateStore{
			templates: make(map[string]FollowUpSagaTemplate),
		}
	})
	return templateStoreInstance
}

// Store stores the template information for later use when character created event is received
func (s *FollowUpSagaTemplateStore) Store(tenantId uuid.UUID, characterName string, input RestModel, template template.RestModel, characterCreationTransactionId uuid.UUID) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Store with tenant-aware key to avoid conflicts
	key := fmt.Sprintf("%s:%s", tenantId.String(), characterName)
	s.templates[key] = FollowUpSagaTemplate{
		TenantId:                       tenantId,
		Input:                          input,
		Template:                       template,
		CharacterCreationTransactionId: characterCreationTransactionId,
	}

	return nil
}

// Get retrieves the stored template information
func (s *FollowUpSagaTemplateStore) Get(tenantId uuid.UUID, characterName string) (FollowUpSagaTemplate, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	key := fmt.Sprintf("%s:%s", tenantId.String(), characterName)
	template, exists := s.templates[key]
	return template, exists
}

// Remove removes the stored template information after use
func (s *FollowUpSagaTemplateStore) Remove(tenantId uuid.UUID, characterName string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	key := fmt.Sprintf("%s:%s", tenantId.String(), characterName)
	delete(s.templates, key)
}

// Clear removes all stored templates (useful for testing)
func (s *FollowUpSagaTemplateStore) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.templates = make(map[string]FollowUpSagaTemplate)
}

// Size returns the number of stored templates
func (s *FollowUpSagaTemplateStore) Size() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return len(s.templates)
}

// storeFollowUpSagaTemplate stores the template information for later use when character created event is received
func storeFollowUpSagaTemplate(ctx context.Context, characterName string, input RestModel, template template.RestModel, characterCreationTransactionId uuid.UUID) error {
	t := tenant.MustFromContext(ctx)
	store := GetFollowUpSagaTemplateStore()
	return store.Store(t.Id(), characterName, input, template, characterCreationTransactionId)
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
	TenantId                       uuid.UUID
	AccountId                      uint32
	CharacterId                    uint32
	CharacterCreationTransactionId uuid.UUID
	FollowUpSagaTransactionId      uuid.UUID
	CharacterCreationCompleted     bool
	FollowUpSagaCompleted          bool
}

// SagaCompletionTrackerStore provides thread-safe storage for saga completion tracking
type SagaCompletionTrackerStore struct {
	trackers map[uuid.UUID]*SagaCompletionTracker
	mutex    sync.RWMutex
}

// Singleton instance for saga completion tracking
var (
	sagaTrackerStoreInstance *SagaCompletionTrackerStore
	sagaTrackerStoreOnce     sync.Once
)

// GetSagaCompletionTrackerStore returns the singleton instance of the saga completion tracker store
func GetSagaCompletionTrackerStore() *SagaCompletionTrackerStore {
	sagaTrackerStoreOnce.Do(func() {
		sagaTrackerStoreInstance = &SagaCompletionTrackerStore{
			trackers: make(map[uuid.UUID]*SagaCompletionTracker),
		}
	})
	return sagaTrackerStoreInstance
}

// StoreTrackerForCharacterCreation stores tracking information for character creation saga
func (s *SagaCompletionTrackerStore) StoreTrackerForCharacterCreation(tenantId uuid.UUID, accountId uint32, characterCreationTransactionId uuid.UUID) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.trackers[characterCreationTransactionId] = &SagaCompletionTracker{
		TenantId:                       tenantId,
		AccountId:                      accountId,
		CharacterCreationTransactionId: characterCreationTransactionId,
		CharacterCreationCompleted:     false,
		FollowUpSagaCompleted:          false,
	}
}

// UpdateTrackerForFollowUpSaga updates the tracker with follow-up saga information
func (s *SagaCompletionTrackerStore) UpdateTrackerForFollowUpSaga(characterCreationTransactionId uuid.UUID, followUpSagaTransactionId uuid.UUID, characterId uint32) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if tracker, exists := s.trackers[characterCreationTransactionId]; exists {
		tracker.FollowUpSagaTransactionId = followUpSagaTransactionId
		tracker.CharacterId = characterId
		// Also store the tracker by follow-up saga transaction ID for easy lookup
		s.trackers[followUpSagaTransactionId] = tracker
	}
}

// MarkSagaCompleted marks a saga as completed and returns the tracker if both sagas are now complete
func (s *SagaCompletionTrackerStore) MarkSagaCompleted(transactionId uuid.UUID) (*SagaCompletionTracker, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if tracker, exists := s.trackers[transactionId]; exists {
		// Check if this is the character creation saga or follow-up saga
		if transactionId == tracker.CharacterCreationTransactionId {
			tracker.CharacterCreationCompleted = true
		} else if transactionId == tracker.FollowUpSagaTransactionId {
			tracker.FollowUpSagaCompleted = true
		}

		// Check if both sagas are now complete
		if tracker.CharacterCreationCompleted && tracker.FollowUpSagaCompleted {
			// Remove both tracker entries since we're done
			delete(s.trackers, tracker.CharacterCreationTransactionId)
			if tracker.FollowUpSagaTransactionId != uuid.Nil {
				delete(s.trackers, tracker.FollowUpSagaTransactionId)
			}
			return tracker, true
		}
	}

	return nil, false
}

// Clear removes all stored trackers (useful for testing)
func (s *SagaCompletionTrackerStore) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.trackers = make(map[uuid.UUID]*SagaCompletionTracker)
}

// Size returns the number of stored trackers
func (s *SagaCompletionTrackerStore) Size() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return len(s.trackers)
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
