package commodities

import (
	"errors"

	"github.com/google/uuid"
)

// ErrInvalidId is returned when the id is invalid (zero UUID)
var ErrInvalidId = errors.New("id must not be zero UUID")

// modelBuilder is used to build Model instances
type modelBuilder struct {
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

// NewModelBuilder creates a new modelBuilder
func NewModelBuilder() *modelBuilder {
	return &modelBuilder{}
}

// CloneModel creates a new modelBuilder with values from the given Model
func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
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
func (b *modelBuilder) SetId(id uuid.UUID) *modelBuilder {
	b.id = id
	return b
}

// SetTemplateId sets the templateId for the modelBuilder
func (b *modelBuilder) SetTemplateId(templateId uint32) *modelBuilder {
	b.templateId = templateId
	return b
}

// SetMesoPrice sets the mesoPrice for the modelBuilder
func (b *modelBuilder) SetMesoPrice(mesoPrice uint32) *modelBuilder {
	b.mesoPrice = mesoPrice
	return b
}

// SetDiscountRate sets the discountRate for the modelBuilder
func (b *modelBuilder) SetDiscountRate(discountRate byte) *modelBuilder {
	b.discountRate = discountRate
	return b
}

// SetTokenTemplateId sets the tokenTemplateId for the modelBuilder
func (b *modelBuilder) SetTokenTemplateId(tokenTemplateId uint32) *modelBuilder {
	b.tokenTemplateId = tokenTemplateId
	return b
}

// SetTokenPrice sets the tokenPrice for the modelBuilder
func (b *modelBuilder) SetTokenPrice(tokenPrice uint32) *modelBuilder {
	b.tokenPrice = tokenPrice
	return b
}

// SetPeriod sets the period for the modelBuilder
func (b *modelBuilder) SetPeriod(period uint32) *modelBuilder {
	b.period = period
	return b
}

// SetLevelLimit sets the levelLimit for the modelBuilder
func (b *modelBuilder) SetLevelLimit(levelLimit uint32) *modelBuilder {
	b.levelLimit = levelLimit
	return b
}

// SetUnitPrice sets the unitPrice for the modelBuilder
func (b *modelBuilder) SetUnitPrice(unitPrice float64) *modelBuilder {
	b.unitPrice = unitPrice
	return b
}

// SetSlotMax sets the slotMax for the modelBuilder
func (b *modelBuilder) SetSlotMax(slotMax uint32) *modelBuilder {
	b.slotMax = slotMax
	return b
}

// Build creates a new Model instance with the builder's values
func (b *modelBuilder) Build() (Model, error) {
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
func (b *modelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
