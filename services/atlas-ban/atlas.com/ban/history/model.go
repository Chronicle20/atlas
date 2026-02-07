package history

import (
	"time"

	"github.com/google/uuid"
)

type Model struct {
	tenantId      uuid.UUID
	id            uint64
	accountId     uint32
	accountName   string
	ipAddress     string
	hwid          string
	success       bool
	failureReason string
	createdAt     time.Time
}

func (m Model) TenantId() uuid.UUID {
	return m.tenantId
}

func (m Model) Id() uint64 {
	return m.id
}

func (m Model) AccountId() uint32 {
	return m.accountId
}

func (m Model) AccountName() string {
	return m.accountName
}

func (m Model) IPAddress() string {
	return m.ipAddress
}

func (m Model) HWID() string {
	return m.hwid
}

func (m Model) Success() bool {
	return m.success
}

func (m Model) FailureReason() string {
	return m.failureReason
}

func (m Model) CreatedAt() time.Time {
	return m.createdAt
}
