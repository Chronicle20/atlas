package commodities

import (
	"github.com/google/uuid"
)

type Model struct {
	id                uuid.UUID
	templateId        uint32
	mesoPrice         uint32
	perfectPitchPrice uint32
}

// Id returns a pointer to the model's id
func (m *Model) Id() *uuid.UUID {
	return &m.id
}

// TemplateId returns a pointer to the model's templateId
func (m *Model) TemplateId() *uint32 {
	return &m.templateId
}

// MesoPrice returns a pointer to the model's mesoPrice
func (m *Model) MesoPrice() *uint32 {
	return &m.mesoPrice
}

// PerfectPitchPrice returns a pointer to the model's perfectPitchPrice
func (m *Model) PerfectPitchPrice() *uint32 {
	return &m.perfectPitchPrice
}

// ModelBuilder is used to build Model instances
type ModelBuilder struct {
	id                uuid.UUID
	templateId        uint32
	mesoPrice         uint32
	perfectPitchPrice uint32
}

// SetId sets the id for the ModelBuilder
func (b *ModelBuilder) SetId(id uuid.UUID) *ModelBuilder {
	b.id = id
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

// SetPerfectPitchPrice sets the perfectPitchPrice for the ModelBuilder
func (b *ModelBuilder) SetPerfectPitchPrice(perfectPitchPrice uint32) *ModelBuilder {
	b.perfectPitchPrice = perfectPitchPrice
	return b
}

// Build creates a new Model instance with the builder's values
func (b *ModelBuilder) Build() Model {
	return Model{
		id:                b.id,
		templateId:        b.templateId,
		mesoPrice:         b.mesoPrice,
		perfectPitchPrice: b.perfectPitchPrice,
	}
}

// Clone creates a new ModelBuilder with values from the given Model
func Clone(model Model) *ModelBuilder {
	return &ModelBuilder{
		id:                model.id,
		templateId:        model.templateId,
		mesoPrice:         model.mesoPrice,
		perfectPitchPrice: model.perfectPitchPrice,
	}
}
