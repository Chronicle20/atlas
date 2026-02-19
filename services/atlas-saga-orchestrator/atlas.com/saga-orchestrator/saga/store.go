package saga

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	tenant "github.com/Chronicle20/atlas-tenant"
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
	ten map[uuid.UUID]tenant.Model // tracks tenant info per transaction for Put
}

// NewPostgresStore creates a new PostgreSQL-backed saga store
func NewPostgresStore(db *gorm.DB, l logrus.FieldLogger) *PostgresStore {
	return &PostgresStore{
		db:  db,
		l:   l,
		ver: make(map[uuid.UUID]int),
		ten: make(map[uuid.UUID]tenant.Model),
	}
}

// GetAll returns all active sagas for a tenant
func (s *PostgresStore) GetAll(tenantId uuid.UUID) []Saga {
	var entities []Entity
	err := s.db.Where("tenant_id = ? AND status IN ?", tenantId, []string{"active", "compensating"}).Find(&entities).Error
	if err != nil {
		s.l.WithError(err).WithField("tenant_id", tenantId.String()).Error("Failed to query sagas")
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

// GetById returns a saga by its transaction ID for a tenant
func (s *PostgresStore) GetById(tenantId uuid.UUID, transactionId uuid.UUID) (Saga, bool) {
	var e Entity
	err := s.db.Where("transaction_id = ? AND tenant_id = ?", transactionId, tenantId).First(&e).Error
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

// Put adds or updates a saga in the store for a tenant.
// Returns VersionConflictError if another instance updated the saga concurrently.
func (s *PostgresStore) Put(tenantId uuid.UUID, saga Saga) error {
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
	t, hasTenant := s.ten[saga.TransactionId()]
	s.mu.RUnlock()

	if hasVer {
		// Optimistic update: only succeed if version matches
		result := s.db.Model(&Entity{}).
			Where("transaction_id = ? AND tenant_id = ? AND version = ?", saga.TransactionId(), tenantId, ver).
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
		// New saga — insert with timeout
		timeoutAt := time.Now().Add(defaultTimeout)

		var tenantRegion string
		var tenantMajor, tenantMinor uint16
		if hasTenant {
			tenantRegion = t.Region()
			tenantMajor = t.MajorVersion()
			tenantMinor = t.MinorVersion()
		}

		e := Entity{
			TransactionId: saga.TransactionId(),
			TenantId:      tenantId,
			TenantRegion:  tenantRegion,
			TenantMajor:   tenantMajor,
			TenantMinor:   tenantMinor,
			SagaType:      string(saga.SagaType()),
			InitiatedBy:   saga.InitiatedBy(),
			Status:        sagaStatus,
			SagaData:      data,
			Version:       1,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			TimeoutAt:     &timeoutAt,
		}

		result := s.db.Clauses(clause.OnConflict{
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

// Remove marks a saga as completed (soft delete) for a tenant
func (s *PostgresStore) Remove(tenantId uuid.UUID, transactionId uuid.UUID) bool {
	result := s.db.Model(&Entity{}).
		Where("transaction_id = ? AND tenant_id = ?", transactionId, tenantId).
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
	delete(s.ten, transactionId)
	s.mu.Unlock()

	return result.RowsAffected > 0
}

// TrackTenant stores the full tenant model for a transaction so Put can persist tenant fields
func (s *PostgresStore) TrackTenant(transactionId uuid.UUID, t tenant.Model) {
	s.mu.Lock()
	s.ten[transactionId] = t
	s.mu.Unlock()
}

// GetAllActive returns all active and compensating sagas across all tenants (for startup recovery)
func (s *PostgresStore) GetAllActive() []Entity {
	var entities []Entity
	err := s.db.Where("status IN ?", []string{"active", "compensating"}).Find(&entities).Error
	if err != nil {
		s.l.WithError(err).Error("Failed to query active sagas for recovery")
		return nil
	}
	return entities
}

// GetTimedOut returns sagas that have exceeded their timeout, locking them for processing
func (s *PostgresStore) GetTimedOut() []Entity {
	var entities []Entity
	err := s.db.
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
func (s *PostgresStore) UpdateStatusFailed(tenantId uuid.UUID, transactionId uuid.UUID) {
	s.db.Model(&Entity{}).
		Where("transaction_id = ? AND tenant_id = ?", transactionId, tenantId).
		Updates(map[string]interface{}{
			"status":     "failed",
			"updated_at": time.Now(),
		})

	// Clean up version tracking
	s.mu.Lock()
	delete(s.ver, transactionId)
	delete(s.ten, transactionId)
	s.mu.Unlock()
}

func entityToSaga(e Entity) (Saga, error) {
	var saga Saga
	err := json.Unmarshal(e.SagaData, &saga)
	if err != nil {
		return Saga{}, err
	}
	return saga, nil
}
