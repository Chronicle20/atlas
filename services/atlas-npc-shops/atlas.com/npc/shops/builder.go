package shops

import (
	"atlas-npc/commodities"
	"errors"
)

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
	recharger   bool
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

// AddCommodity adds a single commodity to the ModelBuilder
func (b *ModelBuilder) AddCommodity(commodity commodities.Model) *ModelBuilder {
	b.commodities = append(b.commodities, commodity)
	return b
}

// SetRecharger sets whether rechargeables can be recharged at this shop
func (b *ModelBuilder) SetRecharger(recharger bool) *ModelBuilder {
	b.recharger = recharger
	return b
}

// Build creates a new Model instance with the builder's values
func (b *ModelBuilder) Build() (Model, error) {
	if b.npcId == 0 {
		return Model{}, errors.New("npcId is required")
	}
	return Model{
		npcId:       b.npcId,
		commodities: b.commodities,
		recharger:   b.recharger,
	}, nil
}

// Clone creates a new ModelBuilder with values from the given Model
func Clone(model Model) *ModelBuilder {
	return &ModelBuilder{
		npcId:       model.npcId,
		commodities: model.commodities,
		recharger:   model.recharger,
	}
}
