package bid

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Builder constructs an immutable bid Model. The id is assigned at create time
// in the administrator, so it is not required here.
type Builder struct {
	id          uuid.UUID
	tenantId    uuid.UUID
	listingId   uuid.UUID
	bidderId    uint32
	amount      uint32
	escrowTxnId uuid.UUID
	state       State
	createdAt   time.Time
}

func NewBuilder(tenantId uuid.UUID, listingId uuid.UUID, bidderId uint32) *Builder {
	return &Builder{tenantId: tenantId, listingId: listingId, bidderId: bidderId}
}

func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetListingId(listingId uuid.UUID) *Builder {
	b.listingId = listingId
	return b
}

func (b *Builder) SetBidderId(bidderId uint32) *Builder {
	b.bidderId = bidderId
	return b
}

func (b *Builder) SetAmount(amount uint32) *Builder {
	b.amount = amount
	return b
}

func (b *Builder) SetEscrowTxnId(escrowTxnId uuid.UUID) *Builder {
	b.escrowTxnId = escrowTxnId
	return b
}

func (b *Builder) SetState(s State) *Builder {
	b.state = s
	return b
}

func (b *Builder) SetCreatedAt(v time.Time) *Builder {
	b.createdAt = v
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.tenantId == uuid.Nil {
		return Model{}, errors.New("tenantId cannot be nil")
	}
	return Model{
		id:          b.id,
		tenantId:    b.tenantId,
		listingId:   b.listingId,
		bidderId:    b.bidderId,
		amount:      b.amount,
		escrowTxnId: b.escrowTxnId,
		state:       b.state,
		createdAt:   b.createdAt,
	}, nil
}
