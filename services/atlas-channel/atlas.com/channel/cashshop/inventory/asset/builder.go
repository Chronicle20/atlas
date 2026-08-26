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

// modelBuilder is a builder for the Model
type modelBuilder struct {
	id               uint32
	compartmentId    uuid.UUID
	item             item.Model
	giftFrom         string
	giftMessage      string
	giftAcknowledged bool
}

// NewModelBuilder creates a new modelBuilder with required fields
func NewModelBuilder(id uint32, compartmentId uuid.UUID, i item.Model) *modelBuilder {
	return &modelBuilder{
		id:            id,
		compartmentId: compartmentId,
		item:          i,
	}
}

// CloneModel creates a builder from this model
func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
		id:               m.id,
		compartmentId:    m.compartmentId,
		item:             m.item,
		giftFrom:         m.giftFrom,
		giftMessage:      m.giftMessage,
		giftAcknowledged: m.giftAcknowledged,
	}
}

// SetId sets the id for this builder
func (b *modelBuilder) SetId(id uint32) *modelBuilder {
	b.id = id
	return b
}

// SetCompartmentId sets the compartmentId for this builder
func (b *modelBuilder) SetCompartmentId(compartmentId uuid.UUID) *modelBuilder {
	b.compartmentId = compartmentId
	return b
}

// SetItem sets the item associated with this asset
func (b *modelBuilder) SetItem(i item.Model) *modelBuilder {
	b.item = i
	return b
}

// SetGiftFrom sets the sender's character name for this gifted asset
func (b *modelBuilder) SetGiftFrom(giftFrom string) *modelBuilder {
	b.giftFrom = giftFrom
	return b
}

// SetGiftMessage sets the sender's message for this gifted asset
func (b *modelBuilder) SetGiftMessage(giftMessage string) *modelBuilder {
	b.giftMessage = giftMessage
	return b
}

// SetGiftAcknowledged sets whether the gift list carrying this asset has
// already been presented to the recipient (task-240 Defect H).
func (b *modelBuilder) SetGiftAcknowledged(giftAcknowledged bool) *modelBuilder {
	b.giftAcknowledged = giftAcknowledged
	return b
}

// Build creates a Model from this builder
func (b *modelBuilder) Build() (Model, error) {
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
	}, nil
}

// MustBuild creates a Model from this builder and panics if validation fails
func (b *modelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
