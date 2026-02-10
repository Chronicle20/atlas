package ban

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrCannotExpirePermanentBan = errors.New("cannot expire a permanent ban")

type BanType byte

const (
	BanTypeIP      BanType = 0
	BanTypeHWID    BanType = 1
	BanTypeAccount BanType = 2
)

type Model struct {
	tenantId   uuid.UUID
	id         uint32
	banType    BanType
	value      string
	reason     string
	reasonCode byte
	permanent  bool
	expiresAt  time.Time
	issuedBy   string
	createdAt  time.Time
	updatedAt  time.Time
}

func (m Model) TenantId() uuid.UUID {
	return m.tenantId
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Type() BanType {
	return m.banType
}

func (m Model) Value() string {
	return m.value
}

func (m Model) Reason() string {
	return m.reason
}

func (m Model) ReasonCode() byte {
	return m.reasonCode
}

func (m Model) Permanent() bool {
	return m.permanent
}

func (m Model) ExpiresAt() time.Time {
	return m.expiresAt
}

func (m Model) IssuedBy() string {
	return m.issuedBy
}

func (m Model) CreatedAt() time.Time {
	return m.createdAt
}

func (m Model) UpdatedAt() time.Time {
	return m.updatedAt
}

func IsExpired(m Model) bool {
	if m.permanent {
		return false
	}
	return !m.expiresAt.IsZero() && time.Now().After(m.expiresAt)
}
