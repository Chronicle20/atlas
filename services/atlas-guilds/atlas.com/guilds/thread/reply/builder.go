package reply

import (
	"errors"
	"time"
)

// Builder provides fluent construction of reply models
type Builder struct {
	id        *uint32
	posterId  *uint32
	message   *string
	createdAt *time.Time
}

// NewBuilder creates a new builder with required parameters
func NewBuilder(id uint32, posterId uint32, message string) *Builder {
	return &Builder{
		id:       &id,
		posterId: &posterId,
		message:  &message,
	}
}

// SetCreatedAt sets the creation timestamp
func (b *Builder) SetCreatedAt(createdAt time.Time) *Builder {
	b.createdAt = &createdAt
	return b
}

// Build validates invariants and constructs the final immutable model
func (b *Builder) Build() (Model, error) {
	if b.id == nil {
		return Model{}, errors.New("reply ID is required")
	}
	if *b.id == 0 {
		return Model{}, errors.New("reply ID must be greater than 0")
	}
	if b.posterId == nil {
		return Model{}, errors.New("poster ID is required")
	}
	if *b.posterId == 0 {
		return Model{}, errors.New("poster ID must be greater than 0")
	}
	if b.message == nil {
		return Model{}, errors.New("reply message is required")
	}

	// Default optional values
	createdAt := time.Now()
	if b.createdAt != nil {
		createdAt = *b.createdAt
	}

	return Model{
		id:        *b.id,
		posterId:  *b.posterId,
		message:   *b.message,
		createdAt: createdAt,
	}, nil
}

// Builder returns a builder initialized with the current model's values
func (m Model) Builder() *Builder {
	// Create value copies to preserve immutability of the original model
	id := m.id
	posterId := m.posterId
	message := m.message
	createdAt := m.createdAt

	return &Builder{
		id:        &id,
		posterId:  &posterId,
		message:   &message,
		createdAt: &createdAt,
	}
}
