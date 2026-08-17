package pending_change

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type modelBuilder struct {
	id                 uuid.UUID
	characterId        uint32
	changeType         string
	status             string
	requestedName      string
	destinationWorldId world.Id
	sourceWorldId      world.Id
	assetId            uint32
	hasAsset           bool
	reason             string
	transactionId      uuid.UUID
	createdAt          time.Time
	expiresAt          time.Time
	resolvedAt         *time.Time
	notifiedAt         *time.Time
}

func NewBuilder() *modelBuilder {
	return &modelBuilder{}
}

func (b *modelBuilder) SetId(id uuid.UUID) *modelBuilder {
	b.id = id
	return b
}

func (b *modelBuilder) SetCharacterId(characterId uint32) *modelBuilder {
	b.characterId = characterId
	return b
}

func (b *modelBuilder) SetType(changeType string) *modelBuilder {
	b.changeType = changeType
	return b
}

func (b *modelBuilder) SetStatus(status string) *modelBuilder {
	b.status = status
	return b
}

func (b *modelBuilder) SetRequestedName(requestedName string) *modelBuilder {
	b.requestedName = requestedName
	return b
}

func (b *modelBuilder) SetDestinationWorldId(destinationWorldId world.Id) *modelBuilder {
	b.destinationWorldId = destinationWorldId
	return b
}

func (b *modelBuilder) SetSourceWorldId(sourceWorldId world.Id) *modelBuilder {
	b.sourceWorldId = sourceWorldId
	return b
}

func (b *modelBuilder) SetAssetId(assetId uint32) *modelBuilder {
	b.assetId = assetId
	b.hasAsset = true
	return b
}

func (b *modelBuilder) SetReason(reason string) *modelBuilder {
	b.reason = reason
	return b
}

func (b *modelBuilder) SetTransactionId(transactionId uuid.UUID) *modelBuilder {
	b.transactionId = transactionId
	return b
}

func (b *modelBuilder) SetCreatedAt(createdAt time.Time) *modelBuilder {
	b.createdAt = createdAt
	return b
}

func (b *modelBuilder) SetExpiresAt(expiresAt time.Time) *modelBuilder {
	b.expiresAt = expiresAt
	return b
}

func (b *modelBuilder) SetResolvedAt(resolvedAt *time.Time) *modelBuilder {
	b.resolvedAt = resolvedAt
	return b
}

func (b *modelBuilder) SetNotifiedAt(notifiedAt *time.Time) *modelBuilder {
	b.notifiedAt = notifiedAt
	return b
}

func (b *modelBuilder) Build() Model {
	return Model{
		id:                 b.id,
		characterId:        b.characterId,
		changeType:         b.changeType,
		status:             b.status,
		requestedName:      b.requestedName,
		destinationWorldId: b.destinationWorldId,
		sourceWorldId:      b.sourceWorldId,
		assetId:            b.assetId,
		hasAsset:           b.hasAsset,
		reason:             b.reason,
		transactionId:      b.transactionId,
		createdAt:          b.createdAt,
		expiresAt:          b.expiresAt,
		resolvedAt:         b.resolvedAt,
		notifiedAt:         b.notifiedAt,
	}
}
