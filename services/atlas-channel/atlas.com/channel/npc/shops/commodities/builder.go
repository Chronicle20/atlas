package commodities

import (
	"errors"

	"github.com/google/uuid"
)

// ErrInvalidId is returned when the id is invalid (zero UUID)
var ErrInvalidId = errors.New("id must not be zero UUID")

// modelBuilder is used to build Model instances
type builder struct {
	id              uuid.UUID
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

// NewBuilder creates a new builder
func NewBuilder() *builder {
	return &builder{}
}

// CloneModel creates a new modelBuilder with values from the given Model
func CloneModel(m Model) *builder {
	return &builder{
		id:              m.id,
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

// SetId sets the id for the modelBuilder
func (b *builder) SetId(id uuid.UUID) *builder {
	b.id = id
	return b
}

// SetTemplateId sets the templateId for the modelBuilder
func (b *builder) SetTemplateId(templateId uint32) *builder {
	b.templateId = templateId
	return b
}

// SetMesoPrice sets the mesoPrice for the modelBuilder
func (b *builder) SetMesoPrice(mesoPrice uint32) *builder {
	b.mesoPrice = mesoPrice
	return b
}

// SetDiscountRate sets the discountRate for the modelBuilder
func (b *builder) SetDiscountRate(discountRate byte) *builder {
	b.discountRate = discountRate
	return b
}

// SetTokenTemplateId sets the tokenTemplateId for the modelBuilder
func (b *builder) SetTokenTemplateId(tokenTemplateId uint32) *builder {
	b.tokenTemplateId = tokenTemplateId
	return b
}

// SetTokenPrice sets the tokenPrice for the modelBuilder
func (b *builder) SetTokenPrice(tokenPrice uint32) *builder {
	b.tokenPrice = tokenPrice
	return b
}

// SetPeriod sets the period for the modelBuilder
func (b *builder) SetPeriod(period uint32) *builder {
	b.period = period
	return b
}

// SetLevelLimit sets the levelLimit for the modelBuilder
func (b *builder) SetLevelLimit(levelLimit uint32) *builder {
	b.levelLimit = levelLimit
	return b
}

// SetUnitPrice sets the unitPrice for the modelBuilder
func (b *builder) SetUnitPrice(unitPrice float64) *builder {
	b.unitPrice = unitPrice
	return b
}

// SetSlotMax sets the slotMax for the modelBuilder
func (b *builder) SetSlotMax(slotMax uint32) *builder {
	b.slotMax = slotMax
	return b
}

// Build creates a new Model instance with the builder's values
func (b *builder) Build() (Model, error) {
	if b.id == uuid.Nil {
		return Model{}, ErrInvalidId
	}
	return Model{
		id:              b.id,
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

// MustBuild creates a new Model instance and panics if validation fails
func (b *builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
