package shops

import "atlas-channel/npc/shops/commodities"

type Model struct {
	npcId       uint32
	commodities []commodities.Model
}

// NpcId returns a pointer to the model's npcId
func (m *Model) NpcId() uint32 {
	return m.npcId
}

// Commodities returns a pointer to the model's commodities
func (m *Model) Commodities() []commodities.Model {
	return m.commodities
}

// NewBuilder is used to initialize a new ModelBuilder
func NewBuilder(npcId uint32) *ModelBuilder {
	return &ModelBuilder{
		npcId: npcId,
	}
}

// ModelBuilder is used to build Model instances
type ModelBuilder struct {
	npcId       uint32
	commodities []commodities.Model
}

// SetNpcId sets the npcId for the ModelBuilder
func (b *ModelBuilder) SetNpcId(npcId uint32) *ModelBuilder {
	b.npcId = npcId
	return b
}

// SetCommodities sets the commodities for the ModelBuilder
func (b *ModelBuilder) SetCommodities(commodities []commodities.Model) *ModelBuilder {
	b.commodities = commodities
	return b
}

// Build creates a new Model instance with the builder's values
func (b *ModelBuilder) Build() Model {
	return Model{
		npcId:       b.npcId,
		commodities: b.commodities,
	}
}

// Clone creates a new ModelBuilder with values from the given Model
func Clone(model Model) *ModelBuilder {
	return &ModelBuilder{
		npcId:       model.npcId,
		commodities: model.commodities,
	}
}
