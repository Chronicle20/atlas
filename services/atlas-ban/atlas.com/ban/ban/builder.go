package ban

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Builder struct {
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

func NewBuilder(tenantId uuid.UUID, banType BanType, value string) *Builder {
	return &Builder{
		tenantId:   tenantId,
		banType:    banType,
		value:      value,
		reason:     "",
		reasonCode: 0,
		permanent:  false,
		issuedBy:   "",
	}
}

func (b *Builder) SetId(id uint32) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetReason(reason string) *Builder {
	b.reason = reason
	return b
}

func (b *Builder) SetReasonCode(reasonCode byte) *Builder {
	b.reasonCode = reasonCode
	return b
}

func (b *Builder) SetPermanent(permanent bool) *Builder {
	b.permanent = permanent
	return b
}

func (b *Builder) SetExpiresAt(expiresAt time.Time) *Builder {
	b.expiresAt = expiresAt
	return b
}

func (b *Builder) SetIssuedBy(issuedBy string) *Builder {
	b.issuedBy = issuedBy
	return b
}

func (b *Builder) SetCreatedAt(createdAt time.Time) *Builder {
	b.createdAt = createdAt
	return b
}

func (b *Builder) SetUpdatedAt(updatedAt time.Time) *Builder {
	b.updatedAt = updatedAt
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.value == "" {
		return Model{}, errors.New("value is required")
	}

	return Model{
		tenantId:   b.tenantId,
		id:         b.id,
		banType:    b.banType,
		value:      b.value,
		reason:     b.reason,
		reasonCode: b.reasonCode,
		permanent:  b.permanent,
		expiresAt:  b.expiresAt,
		issuedBy:   b.issuedBy,
		createdAt:  b.createdAt,
		updatedAt:  b.updatedAt,
	}, nil
}
