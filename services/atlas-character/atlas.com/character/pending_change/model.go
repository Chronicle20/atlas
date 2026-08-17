package pending_change

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type Model struct {
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

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) Type() string {
	return m.changeType
}

func (m Model) Status() string {
	return m.status
}

func (m Model) RequestedName() string {
	return m.requestedName
}

func (m Model) DestinationWorldId() world.Id {
	return m.destinationWorldId
}

func (m Model) SourceWorldId() world.Id {
	return m.sourceWorldId
}

func (m Model) AssetId() uint32 {
	return m.assetId
}

func (m Model) HasAsset() bool {
	return m.hasAsset
}

func (m Model) Reason() string {
	return m.reason
}

func (m Model) TransactionId() uuid.UUID {
	return m.transactionId
}

func (m Model) CreatedAt() time.Time {
	return m.createdAt
}

func (m Model) ExpiresAt() time.Time {
	return m.expiresAt
}

func (m Model) ResolvedAt() *time.Time {
	return m.resolvedAt
}

func (m Model) NotifiedAt() *time.Time {
	return m.notifiedAt
}
