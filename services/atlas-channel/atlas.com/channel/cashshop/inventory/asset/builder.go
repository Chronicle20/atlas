package asset

import (
	"atlas-channel/cashshop/item"
	"errors"

	"github.com/google/uuid"
)

// ErrInvalidId is returned when the id is invalid (zero)
var ErrInvalidId = errors.New("id must not be zero")

// ErrInvalidCompartmentId is returned when the compartmentId is invalid (zero UUID)
var ErrInvalidCompartmentId = errors.New("compartmentId must not be zero UUID")

// builder is a builder for the Model
type builder struct {
	id               uint32
	compartmentId    uuid.UUID
	item             item.Model
	giftFrom         string
	giftMessage      string
	giftAcknowledged bool
	giftNoteSent     bool
}

// NewBuilder creates a new builder with required fields
func NewBuilder(id uint32, compartmentId uuid.UUID, i item.Model) *builder {
	return &builder{
		id:            id,
		compartmentId: compartmentId,
		item:          i,
	}
}

// CloneModel creates a builder from this model
func CloneModel(m Model) *builder {
	return &builder{
		id:               m.id,
		compartmentId:    m.compartmentId,
		item:             m.item,
		giftFrom:         m.giftFrom,
		giftMessage:      m.giftMessage,
		giftAcknowledged: m.giftAcknowledged,
		giftNoteSent:     m.giftNoteSent,
	}
}

// SetId sets the id for this builder
func (b *builder) SetId(id uint32) *builder {
	b.id = id
	return b
}

// SetCompartmentId sets the compartmentId for this builder
func (b *builder) SetCompartmentId(compartmentId uuid.UUID) *builder {
	b.compartmentId = compartmentId
	return b
}

// SetItem sets the item associated with this asset
func (b *builder) SetItem(i item.Model) *builder {
	b.item = i
	return b
}

// SetGiftFrom sets the sender's character name for this gifted asset
func (b *builder) SetGiftFrom(giftFrom string) *builder {
	b.giftFrom = giftFrom
	return b
}

// SetGiftMessage sets the sender's message for this gifted asset
func (b *builder) SetGiftMessage(giftMessage string) *builder {
	b.giftMessage = giftMessage
	return b
}

// SetGiftAcknowledged sets whether the gift list carrying this asset has
// already been presented to the recipient (task-240 Defect H).
func (b *builder) SetGiftAcknowledged(giftAcknowledged bool) *builder {
	b.giftAcknowledged = giftAcknowledged
	return b
}

// SetGiftNoteSent sets whether the gift-forward note for this asset has
// already been sent to the gifter (task-240 Defect I).
func (b *builder) SetGiftNoteSent(giftNoteSent bool) *builder {
	b.giftNoteSent = giftNoteSent
	return b
}

// Build creates a Model from this builder
func (b *builder) Build() (Model, error) {
	if b.id == 0 {
		return Model{}, ErrInvalidId
	}
	if b.compartmentId == uuid.Nil {
		return Model{}, ErrInvalidCompartmentId
	}
	return Model{
		id:               b.id,
		compartmentId:    b.compartmentId,
		item:             b.item,
		giftFrom:         b.giftFrom,
		giftMessage:      b.giftMessage,
		giftAcknowledged: b.giftAcknowledged,
		giftNoteSent:     b.giftNoteSent,
	}, nil
}

// MustBuild creates a Model from this builder and panics if validation fails
func (b *builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
