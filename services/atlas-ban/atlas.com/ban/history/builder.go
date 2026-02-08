package history

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Builder struct {
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

func NewBuilder(tenantId uuid.UUID, accountId uint32, accountName string) *Builder {
	return &Builder{
		tenantId:    tenantId,
		accountId:   accountId,
		accountName: accountName,
		success:     false,
	}
}

func (b *Builder) SetId(id uint64) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetIPAddress(ipAddress string) *Builder {
	b.ipAddress = ipAddress
	return b
}

func (b *Builder) SetHWID(hwid string) *Builder {
	b.hwid = hwid
	return b
}

func (b *Builder) SetSuccess(success bool) *Builder {
	b.success = success
	return b
}

func (b *Builder) SetFailureReason(failureReason string) *Builder {
	b.failureReason = failureReason
	return b
}

func (b *Builder) SetCreatedAt(createdAt time.Time) *Builder {
	b.createdAt = createdAt
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.accountId == 0 {
		return Model{}, errors.New("accountId is required")
	}

	return Model{
		tenantId:      b.tenantId,
		id:            b.id,
		accountId:     b.accountId,
		accountName:   b.accountName,
		ipAddress:     b.ipAddress,
		hwid:          b.hwid,
		success:       b.success,
		failureReason: b.failureReason,
		createdAt:     b.createdAt,
	}, nil
}
