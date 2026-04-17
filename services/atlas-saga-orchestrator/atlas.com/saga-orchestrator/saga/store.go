package saga

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var defaultTimeout = 5 * time.Minute

// SetDefaultTimeout sets the default timeout duration for new sagas
func SetDefaultTimeout(d time.Duration) {
	defaultTimeout = d
}

// VersionConflictError is returned when an optimistic locking conflict occurs
type VersionConflictError struct {
	TransactionId uuid.UUID
}

func (e *VersionConflictError) Error() string {
	return "version conflict for saga " + e.TransactionId.String()
}

// PostgresStore implements the Cache interface backed by PostgreSQL
type PostgresStore struct {
	db  *gorm.DB
	l   logrus.FieldLogger
	mu  sync.RWMutex
	ver map[uuid.UUID]int // tracks last-read version per transaction
}

// NewPostgresStore creates a new PostgreSQL-backed saga store
func NewPostgresStore(db *gorm.DB, l logrus.FieldLogger) *PostgresStore {
	return &PostgresStore{
		db:  db,
		l:   l,
		ver: make(map[uuid.UUID]int),
	}
}

// GetAll returns all active sagas for the tenant in context
func (s *PostgresStore) GetAll(ctx context.Context) []Saga {
	var entities []Entity
	err := s.db.WithContext(ctx).Where("status IN ?", []string{"active", "compensating"}).Find(&entities).Error
	if err != nil {
		s.l.WithError(err).Error("Failed to query sagas")
		return []Saga{}
	}

	result := make([]Saga, 0, len(entities))
	for _, e := range entities {
		saga, err := entityToSaga(e)
		if err != nil {
			s.l.WithError(err).WithField("transaction_id", e.TransactionId.String()).Error("Failed to deserialize saga")
			continue
		}
		result = append(result, saga)
	}
	return result
}

// GetById returns a saga by its transaction ID for the tenant in context
func (s *PostgresStore) GetById(ctx context.Context, transactionId uuid.UUID) (Saga, bool) {
	var e Entity
	err := s.db.WithContext(ctx).Where("transaction_id = ?", transactionId).First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Saga{}, false
		}
		s.l.WithError(err).WithField("transaction_id", transactionId.String()).Error("Failed to query saga")
		return Saga{}, false
	}

	saga, err := entityToSaga(e)
	if err != nil {
		s.l.WithError(err).WithField("transaction_id", transactionId.String()).Error("Failed to deserialize saga")
		return Saga{}, false
	}

	// Track the version we read for optimistic locking
	s.mu.Lock()
	s.ver[transactionId] = e.Version
	s.mu.Unlock()

	return saga, true
}

// Put adds or updates a saga in the store for the tenant in context.
// Returns VersionConflictError if another instance updated the saga concurrently.
func (s *PostgresStore) Put(ctx context.Context, saga Saga) error {
	t := tenant.MustFromContext(ctx)
	tenantId := t.Id()

	data, err := json.Marshal(saga)
	if err != nil {
		s.l.WithError(err).WithField("transaction_id", saga.TransactionId().String()).Error("Failed to serialize saga")
		return err
	}

	sagaStatus := "active"
	if saga.Failing() {
		sagaStatus = "compensating"
	}

	// Check if we have a tracked version (existing saga)
	s.mu.RLock()
	ver, hasVer := s.ver[saga.TransactionId()]
	s.mu.RUnlock()

	if hasVer {
		// Optimistic update: only succeed if version matches
		result := s.db.WithContext(ctx).Model(&Entity{}).
			Where("transaction_id = ? AND version = ?", saga.TransactionId(), ver).
			Updates(map[string]interface{}{
				"saga_type":    string(saga.SagaType()),
				"initiated_by": saga.InitiatedBy(),
				"status":       sagaStatus,
				"saga_data":    data,
				"version":      ver + 1,
				"updated_at":   time.Now(),
			})

		if result.Error != nil {
			s.l.WithError(result.Error).WithField("transaction_id", saga.TransactionId().String()).Error("Failed to update saga")
			return result.Error
		}

		if result.RowsAffected == 0 {
			s.l.WithFields(logrus.Fields{
				"transaction_id": saga.TransactionId().String(),
				"expected_ver":   ver,
			}).Warn("Optimistic locking conflict on saga update")
			// Clear stale version so next GetById re-reads fresh
			s.mu.Lock()
			delete(s.ver, saga.TransactionId())
			s.mu.Unlock()
			return &VersionConflictError{TransactionId: saga.TransactionId()}
		}

		// Update tracked version
		s.mu.Lock()
		s.ver[saga.TransactionId()] = ver + 1
		s.mu.Unlock()
	} else {
		// New saga -- insert with timeout
		timeoutAt := time.Now().Add(defaultTimeout)

		e := Entity{
			TransactionId: saga.TransactionId(),
			TenantId:      tenantId,
			TenantRegion:  t.Region(),
			TenantMajor:   t.MajorVersion(),
			TenantMinor:   t.MinorVersion(),
			SagaType:      string(saga.SagaType()),
			InitiatedBy:   saga.InitiatedBy(),
			Status:        sagaStatus,
			SagaData:      data,
			Version:       1,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			TimeoutAt:     &timeoutAt,
		}

		result := s.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "transaction_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"saga_type", "initiated_by", "status", "saga_data", "version", "updated_at"}),
		}).Create(&e)

		if result.Error != nil {
			s.l.WithError(result.Error).WithField("transaction_id", saga.TransactionId().String()).Error("Failed to insert saga")
			return result.Error
		}

		// Track the version
		s.mu.Lock()
		s.ver[saga.TransactionId()] = 1
		s.mu.Unlock()
	}
	return nil
}

// Remove marks a saga as completed (soft delete) for the tenant in context
func (s *PostgresStore) Remove(ctx context.Context, transactionId uuid.UUID) bool {
	result := s.db.WithContext(ctx).Model(&Entity{}).
		Where("transaction_id = ?", transactionId).
		Updates(map[string]interface{}{
			"status":     "completed",
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		s.l.WithError(result.Error).WithField("transaction_id", transactionId.String()).Error("Failed to remove saga")
		return false
	}

	// Clean up version tracking
	s.mu.Lock()
	delete(s.ver, transactionId)
	s.mu.Unlock()

	return result.RowsAffected > 0
}

// GetAllActive returns all active and compensating sagas across all tenants (for startup recovery)
func (s *PostgresStore) GetAllActive(ctx context.Context) []Entity {
	var entities []Entity
	err := s.db.WithContext(database.WithoutTenantFilter(ctx)).Where("status IN ?", []string{"active", "compensating"}).Find(&entities).Error
	if err != nil {
		s.l.WithError(err).Error("Failed to query active sagas for recovery")
		return nil
	}
	return entities
}

// GetTimedOut returns sagas that have exceeded their timeout, locking them for processing
func (s *PostgresStore) GetTimedOut(ctx context.Context) []Entity {
	var entities []Entity
	err := s.db.WithContext(database.WithoutTenantFilter(ctx)).
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("status = ? AND timeout_at IS NOT NULL AND timeout_at < ?", "active", time.Now()).
		Find(&entities).Error
	if err != nil {
		s.l.WithError(err).Error("Failed to query timed-out sagas")
		return nil
	}
	return entities
}

// UpdateStatusFailed marks a saga as failed
func (s *PostgresStore) UpdateStatusFailed(ctx context.Context, transactionId uuid.UUID) {
	s.db.WithContext(ctx).Model(&Entity{}).
		Where("transaction_id = ?", transactionId).
		Updates(map[string]interface{}{
			"status":     "failed",
			"updated_at": time.Now(),
		})

	// Clean up version tracking
	s.mu.Lock()
	delete(s.ver, transactionId)
	s.mu.Unlock()
}

// lifecycleToStatus maps the in-process lifecycle state to the DB `status`
// column string. The existing column values ("active" / "compensating" /
// "completed" / "failed") are preserved for backward compatibility.
func lifecycleToStatus(l SagaLifecycleState) string {
	switch l {
	case SagaLifecyclePending:
		return "active"
	case SagaLifecycleCompensating:
		return "compensating"
	case SagaLifecycleCompleted:
		return "completed"
	case SagaLifecycleFailed:
		return "failed"
	}
	return ""
}

// statusToLifecycle reverses lifecycleToStatus.
func statusToLifecycle(s string) (SagaLifecycleState, bool) {
	switch s {
	case "active":
		return SagaLifecyclePending, true
	case "compensating":
		return SagaLifecycleCompensating, true
	case "completed":
		return SagaLifecycleCompleted, true
	case "failed":
		return SagaLifecycleFailed, true
	}
	return "", false
}

// TryTransition performs an atomic conditional UPDATE on the saga row: the
// lifecycle moves from `from` to `to` only if the current row status matches.
// Returns false when the saga is missing, when the transition is disallowed
// by IsValidTransition, or when another writer has already changed status.
func (s *PostgresStore) TryTransition(ctx context.Context, transactionId uuid.UUID, from, to SagaLifecycleState) bool {
	if !IsValidTransition(from, to) {
		return false
	}
	fromStatus := lifecycleToStatus(from)
	toStatus := lifecycleToStatus(to)
	if fromStatus == "" || toStatus == "" {
		return false
	}
	result := s.db.WithContext(ctx).Model(&Entity{}).
		Where("transaction_id = ? AND status = ?", transactionId, fromStatus).
		Updates(map[string]interface{}{
			"status":     toStatus,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		s.l.WithError(result.Error).WithField("transaction_id", transactionId.String()).Error("TryTransition update failed")
		return false
	}
	return result.RowsAffected > 0
}

// GetLifecycle returns the current lifecycle state of a saga, or (zero, false)
// if the saga does not exist (or the DB-side status is not recognized).
func (s *PostgresStore) GetLifecycle(ctx context.Context, transactionId uuid.UUID) (SagaLifecycleState, bool) {
	var row struct {
		Status string
	}
	err := s.db.WithContext(ctx).Model(&Entity{}).
		Select("status").
		Where("transaction_id = ?", transactionId).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", false
		}
		s.l.WithError(err).WithField("transaction_id", transactionId.String()).Error("GetLifecycle query failed")
		return "", false
	}
	return statusToLifecycle(row.Status)
}

func entityToSaga(e Entity) (Saga, error) {
	var saga Saga
	err := json.Unmarshal(e.SagaData, &saga)
	if err != nil {
		return Saga{}, err
	}
	return saga, nil
}
