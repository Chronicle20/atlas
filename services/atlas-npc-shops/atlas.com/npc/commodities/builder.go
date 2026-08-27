package commodities

import (
	"errors"

	"github.com/google/uuid"
)

// NewBuilder is used to initialize a new Builder
func NewBuilder() *Builder {
	return &Builder{}
}

// Builder is used to build Model instances
type Builder struct {
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

// SetId sets the id for the Builder
func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

// SetNpcId sets the npcId for the Builder
func (b *Builder) SetNpcId(npcId uint32) *Builder {
	b.npcId = npcId
	return b
}

// SetTemplateId sets the templateId for the Builder
func (b *Builder) SetTemplateId(templateId uint32) *Builder {
	b.templateId = templateId
	return b
}

// SetMesoPrice sets the mesoPrice for the Builder
func (b *Builder) SetMesoPrice(mesoPrice uint32) *Builder {
	b.mesoPrice = mesoPrice
	return b
}

// SetDiscountRate sets the discountRate for the Builder
func (b *Builder) SetDiscountRate(discountRate byte) *Builder {
	b.discountRate = discountRate
	return b
}

// SetTokenTemplateId sets the tokenTemplateId for the Builder
func (b *Builder) SetTokenTemplateId(tokenTemplateId uint32) *Builder {
	b.tokenTemplateId = tokenTemplateId
	return b
}

// SetTokenPrice sets the tokenPrice for the Builder
func (b *Builder) SetTokenPrice(tokenPrice uint32) *Builder {
	b.tokenPrice = tokenPrice
	return b
}

// SetPeriod sets the period for the Builder
func (b *Builder) SetPeriod(period uint32) *Builder {
	b.period = period
	return b
}

// SetLevelLimit sets the levelLimit for the Builder
func (b *Builder) SetLevelLimit(levelLimit uint32) *Builder {
	b.levelLimit = levelLimit
	return b
}

// SetUnitPrice sets the unitPrice for the Builder
func (b *Builder) SetUnitPrice(unitPrice float64) *Builder {
	b.unitPrice = unitPrice
	return b
}

// SetSlotMax sets the slotMax for the Builder
func (b *Builder) SetSlotMax(slotMax uint32) *Builder {
	b.slotMax = slotMax
	return b
}

// Build creates a new Model instance with the builder's values
func (b *Builder) Build() (Model, error) {
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

// Clone creates a new Builder with values from the given Model
func Clone(m Model) *Builder {
	return &Builder{
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
