package shops

import (
	"atlas-channel/npc/shops/commodities"
	"errors"
)

// ErrInvalidNpcId is returned when the npcId is invalid (zero)
var ErrInvalidNpcId = errors.New("npcId must be greater than 0")

// modelBuilder is used to build Model instances
type modelBuilder struct {
	npcId       uint32
	commodities []commodities.Model
}

// NewModelBuilder creates a new modelBuilder
func NewModelBuilder() *modelBuilder {
	return &modelBuilder{}
}

// CloneModel creates a new modelBuilder with values from the given Model
func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
		npcId:       m.npcId,
		commodities: m.commodities,
	}
}

// SetNpcId sets the npcId for the modelBuilder
func (b *modelBuilder) SetNpcId(npcId uint32) *modelBuilder {
	b.npcId = npcId
	return b
}

// SetCommodities sets the commodities for the modelBuilder
func (b *modelBuilder) SetCommodities(c []commodities.Model) *modelBuilder {
	b.commodities = c
	return b
}

// Build creates a new Model instance with the builder's values
func (b *modelBuilder) Build() (Model, error) {
	if b.npcId == 0 {
		return Model{}, ErrInvalidNpcId
	}
	return Model{
		npcId:       b.npcId,
		commodities: b.commodities,
	}, nil
}

// MustBuild creates a new Model instance and panics if validation fails
func (b *modelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
