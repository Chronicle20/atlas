package shops

import (
	"atlas-npc/commodities"
	"errors"
)

// NewBuilder is used to initialize a new Builder
func NewBuilder(npcId uint32) *Builder {
	return &Builder{
		npcId: npcId,
	}
}

// Builder is used to build Model instances
type Builder struct {
	npcId       uint32
	commodities []commodities.Model
	recharger   bool
}

// SetNpcId sets the npcId for the Builder
func (b *Builder) SetNpcId(npcId uint32) *Builder {
	b.npcId = npcId
	return b
}

// SetCommodities sets the commodities for the Builder
func (b *Builder) SetCommodities(commodities []commodities.Model) *Builder {
	b.commodities = commodities
	return b
}

// AddCommodity adds a single commodity to the Builder
func (b *Builder) AddCommodity(commodity commodities.Model) *Builder {
	b.commodities = append(b.commodities, commodity)
	return b
}

// SetRecharger sets whether rechargeables can be recharged at this shop
func (b *Builder) SetRecharger(recharger bool) *Builder {
	b.recharger = recharger
	return b
}

// Build creates a new Model instance with the builder's values
func (b *Builder) Build() (Model, error) {
	if b.npcId == 0 {
		return Model{}, errors.New("npcId is required")
	}
	return Model{
		npcId:       b.npcId,
		commodities: b.commodities,
		recharger:   b.recharger,
	}, nil
}

// Clone creates a new Builder with values from the given Model
func Clone(model Model) *Builder {
	return &Builder{
		npcId:       model.npcId,
		commodities: model.commodities,
		recharger:   model.recharger,
	}
}
