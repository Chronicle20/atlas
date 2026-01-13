package title

import (
	"errors"
	"github.com/google/uuid"
)

// Builder provides fluent construction of title models
type Builder struct {
	tenantId *uuid.UUID
	id       *uuid.UUID
	guildId  *uint32
	name     *string
	index    *byte
}

// NewBuilder creates a new builder with required parameters
func NewBuilder(tenantId uuid.UUID, id uuid.UUID, guildId uint32, name string, index byte) *Builder {
	return &Builder{
		tenantId: &tenantId,
		id:       &id,
		guildId:  &guildId,
		name:     &name,
		index:    &index,
	}
}

// SetName sets the title name
func (b *Builder) SetName(name string) *Builder {
	b.name = &name
	return b
}

// SetIndex sets the title index
func (b *Builder) SetIndex(index byte) *Builder {
	b.index = &index
	return b
}

// Build validates invariants and constructs the final immutable model
func (b *Builder) Build() (Model, error) {
	if b.tenantId == nil {
		return Model{}, errors.New("tenant ID is required")
	}
	if b.id == nil {
		return Model{}, errors.New("title ID is required")
	}
	if b.guildId == nil {
		return Model{}, errors.New("guild ID is required")
	}
	if *b.guildId == 0 {
		return Model{}, errors.New("guild ID must be greater than 0")
	}
	if b.name == nil || *b.name == "" {
		return Model{}, errors.New("title name is required")
	}
	if b.index == nil {
		return Model{}, errors.New("title index is required")
	}

	return Model{
		tenantId: *b.tenantId,
		id:       *b.id,
		guildId:  *b.guildId,
		name:     *b.name,
		index:    *b.index,
	}, nil
}

// Builder returns a builder initialized with the current model's values
func (m Model) Builder() *Builder {
	// Create value copies to preserve immutability of the original model
	tenantId := m.tenantId
	id := m.id
	guildId := m.guildId
	name := m.name
	index := m.index

	return &Builder{
		tenantId: &tenantId,
		id:       &id,
		guildId:  &guildId,
		name:     &name,
		index:    &index,
	}
}
