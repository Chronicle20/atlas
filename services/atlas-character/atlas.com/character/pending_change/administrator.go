package pending_change

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	// ErrAlreadyPending maps idx_pc_one_pending_per_type (FR-2.3).
	ErrAlreadyPending = errors.New("request already pending")
	// ErrNameReserved maps idx_pc_name_reservation (FR-3.3).
	ErrNameReserved = errors.New("name reserved")
	// ErrNotFound is returned for an unknown pending-change id.
	ErrNotFound = errors.New("pending change not found")
	// ErrAlreadyTerminal is returned when the caller asked for a transition on
	// a record that has already left PENDING. The REST layer maps it to 409.
	ErrAlreadyTerminal = errors.New("pending change already terminal")
)

func create(db *gorm.DB, tenantId uuid.UUID, m Model) (Model, error) {
	e := &entity{
		Id:            m.Id(),
		TenantId:      tenantId,
		CharacterId:   m.CharacterId(),
		Type:          m.Type(),
		Status:        m.Status(),
		SourceWorldId: m.SourceWorldId(),
		Reason:        m.Reason(),
		TransactionId: m.TransactionId(),
		CreatedAt:     m.CreatedAt(),
		ExpiresAt:     m.ExpiresAt(),
	}
	if n := m.RequestedName(); n != "" {
		lower := strings.ToLower(n)
		e.RequestedName = &n
		e.RequestedNameLower = &lower
	}
	if m.Type() == TypeWorldTransfer {
		d := m.DestinationWorldId()
		e.DestinationWorldId = &d
	}
	if m.HasAsset() {
		a := m.AssetId()
		e.AssetId = &a
	}

	if err := db.Create(e).Error; err != nil {
		return Model{}, mapUniqueViolation(err)
	}
	return modelFromEntity(*e)
}

// mapUniqueViolation turns a partial-unique-index violation into the reason the
// caller reports. Discriminated on the index name, because both indexes are
// 23505 and only the name says which invariant the insert hit.
func mapUniqueViolation(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "idx_pc_one_pending_per_type"):
		return ErrAlreadyPending
	case strings.Contains(msg, "idx_pc_name_reservation"):
		return ErrNameReserved
	default:
		return err
	}
}

// transition moves a record out of PENDING exactly once. The returned bool is
// the idempotency signal for the whole refund path (design §3.10): callers emit
// the refund and the resolved notification ONLY when it is true, so a
// redelivered Kafka command mints nothing.
func transition(db *gorm.DB, id uuid.UUID, status string, reason string, at time.Time) (Model, bool, error) {
	res := db.Model(&entity{}).
		Where("id = ? AND status = ?", id, StatusPending).
		Updates(map[string]interface{}{"status": status, "reason": reason, "resolved_at": at})
	if res.Error != nil {
		return Model{}, false, res.Error
	}
	m, err := getById(db, id)
	if err != nil {
		return Model{}, false, err
	}
	return m, res.RowsAffected == 1, nil
}

func markNotified(db *gorm.DB, id uuid.UUID, at time.Time) error {
	return db.Model(&entity{}).
		Where("id = ? AND notified_at IS NULL", id).
		Update("notified_at", at).Error
}
