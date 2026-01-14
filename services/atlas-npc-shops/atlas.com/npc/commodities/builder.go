package commodities

import (
	"errors"

	"github.com/google/uuid"
)

// NewBuilder is used to initialize a new ModelBuilder
func NewBuilder() *ModelBuilder {
	return &ModelBuilder{}
}

// ModelBuilder is used to build Model instances
type ModelBuilder struct {
	id              uuid.UUID
	npcId           uint32
	templateId      uint32
	mesoPrice       uint32
	discountRate    byte
	tokenTemplateId uint32
	tokenPrice      uint32
	period          uint32
	levelLimit      uint32
	unitPrice       float64
	slotMax         uint32
}

// SetId sets the id for the ModelBuilder
func (b *ModelBuilder) SetId(id uuid.UUID) *ModelBuilder {
	b.id = id
	return b
}

// SetNpcId sets the npcId for the ModelBuilder
func (b *ModelBuilder) SetNpcId(npcId uint32) *ModelBuilder {
	b.npcId = npcId
	return b
}

// SetTemplateId sets the templateId for the ModelBuilder
func (b *ModelBuilder) SetTemplateId(templateId uint32) *ModelBuilder {
	b.templateId = templateId
	return b
}

// SetMesoPrice sets the mesoPrice for the ModelBuilder
func (b *ModelBuilder) SetMesoPrice(mesoPrice uint32) *ModelBuilder {
	b.mesoPrice = mesoPrice
	return b
}

// SetDiscountRate sets the discountRate for the ModelBuilder
func (b *ModelBuilder) SetDiscountRate(discountRate byte) *ModelBuilder {
	b.discountRate = discountRate
	return b
}

// SetTokenTemplateId sets the tokenTemplateId for the ModelBuilder
func (b *ModelBuilder) SetTokenTemplateId(tokenTemplateId uint32) *ModelBuilder {
	b.tokenTemplateId = tokenTemplateId
	return b
}

// SetTokenPrice sets the tokenPrice for the ModelBuilder
func (b *ModelBuilder) SetTokenPrice(tokenPrice uint32) *ModelBuilder {
	b.tokenPrice = tokenPrice
	return b
}

// SetPeriod sets the period for the ModelBuilder
func (b *ModelBuilder) SetPeriod(period uint32) *ModelBuilder {
	b.period = period
	return b
}

// SetLevelLimit sets the levelLimit for the ModelBuilder
func (b *ModelBuilder) SetLevelLimit(levelLimit uint32) *ModelBuilder {
	b.levelLimit = levelLimit
	return b
}

// SetUnitPrice sets the unitPrice for the ModelBuilder
func (b *ModelBuilder) SetUnitPrice(unitPrice float64) *ModelBuilder {
	b.unitPrice = unitPrice
	return b
}

// SetSlotMax sets the slotMax for the ModelBuilder
func (b *ModelBuilder) SetSlotMax(slotMax uint32) *ModelBuilder {
	b.slotMax = slotMax
	return b
}

// Build creates a new Model instance with the builder's values
func (b *ModelBuilder) Build() (Model, error) {
	if b.id == uuid.Nil {
		return Model{}, errors.New("id is required")
	}
	if b.templateId == 0 {
		return Model{}, errors.New("templateId is required")
	}
	return Model{
		id:              b.id,
		npcId:           b.npcId,
		templateId:      b.templateId,
		mesoPrice:       b.mesoPrice,
		discountRate:    b.discountRate,
		tokenTemplateId: b.tokenTemplateId,
		tokenPrice:      b.tokenPrice,
		period:          b.period,
		levelLimit:      b.levelLimit,
		unitPrice:       b.unitPrice,
		slotMax:         b.slotMax,
	}, nil
}

// Clone creates a new ModelBuilder with values from the given Model
func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
		id:              m.id,
		npcId:           m.npcId,
		templateId:      m.templateId,
		mesoPrice:       m.mesoPrice,
		discountRate:    m.discountRate,
		tokenTemplateId: m.tokenTemplateId,
		tokenPrice:      m.tokenPrice,
		period:          m.period,
		levelLimit:      m.levelLimit,
		unitPrice:       m.unitPrice,
		slotMax:         m.slotMax,
	}
}
