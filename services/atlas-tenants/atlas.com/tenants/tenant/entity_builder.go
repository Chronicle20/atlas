package tenant

import "github.com/google/uuid"

type entityBuilder struct {
	id           uuid.UUID
	name         string
	region       string
	majorVersion uint16
	minorVersion uint16
}

// NewEntityBuilder creates a new entity builder
func NewEntityBuilder() *entityBuilder {
	return &entityBuilder{}
}

// SetId sets the tenant ID
func (b *entityBuilder) SetId(id uuid.UUID) *entityBuilder {
	b.id = id
	return b
}

// SetName sets the tenant name
func (b *entityBuilder) SetName(name string) *entityBuilder {
	b.name = name
	return b
}

// SetRegion sets the tenant region
func (b *entityBuilder) SetRegion(region string) *entityBuilder {
	b.region = region
	return b
}

// SetMajorVersion sets the tenant major version
func (b *entityBuilder) SetMajorVersion(majorVersion uint16) *entityBuilder {
	b.majorVersion = majorVersion
	return b
}

// SetMinorVersion sets the tenant minor version
func (b *entityBuilder) SetMinorVersion(minorVersion uint16) *entityBuilder {
	b.minorVersion = minorVersion
	return b
}

// Build creates a new Entity
func (b *entityBuilder) Build() Entity {
	return Entity{
		ID:           b.id,
		Name:         b.name,
		Region:       b.region,
		MajorVersion: b.majorVersion,
		MinorVersion: b.minorVersion,
	}
}

// FromModel creates an Entity from a Model
func FromModel(m Model) Entity {
	return NewEntityBuilder().
		SetId(m.Id()).
		SetName(m.Name()).
		SetRegion(m.Region()).
		SetMajorVersion(m.MajorVersion()).
		SetMinorVersion(m.MinorVersion()).
		Build()
}
