package pending_change

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type builder struct {
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

func NewBuilder() *builder {
	return &builder{}
}

func (b *builder) SetId(id uuid.UUID) *builder {
	b.id = id
	return b
}

func (b *builder) SetCharacterId(characterId uint32) *builder {
	b.characterId = characterId
	return b
}

func (b *builder) SetType(changeType string) *builder {
	b.changeType = changeType
	return b
}

func (b *builder) SetStatus(status string) *builder {
	b.status = status
	return b
}

func (b *builder) SetRequestedName(requestedName string) *builder {
	b.requestedName = requestedName
	return b
}

func (b *builder) SetDestinationWorldId(destinationWorldId world.Id) *builder {
	b.destinationWorldId = destinationWorldId
	return b
}

func (b *builder) SetSourceWorldId(sourceWorldId world.Id) *builder {
	b.sourceWorldId = sourceWorldId
	return b
}

func (b *builder) SetAssetId(assetId uint32) *builder {
	b.assetId = assetId
	b.hasAsset = true
	return b
}

func (b *builder) SetReason(reason string) *builder {
	b.reason = reason
	return b
}

func (b *builder) SetTransactionId(transactionId uuid.UUID) *builder {
	b.transactionId = transactionId
	return b
}

func (b *builder) SetCreatedAt(createdAt time.Time) *builder {
	b.createdAt = createdAt
	return b
}

func (b *builder) SetExpiresAt(expiresAt time.Time) *builder {
	b.expiresAt = expiresAt
	return b
}

func (b *builder) SetResolvedAt(resolvedAt *time.Time) *builder {
	b.resolvedAt = resolvedAt
	return b
}

func (b *builder) SetNotifiedAt(notifiedAt *time.Time) *builder {
	b.notifiedAt = notifiedAt
	return b
}

func (b *builder) Build() Model {
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
