package tenant

import (
	"fmt"
	"github.com/google/uuid"
)

// Model represents a tenant in the domain
type Model struct {
	id           uuid.UUID
	name         string
	region       string
	majorVersion uint16
	minorVersion uint16
}

// Id returns the tenant ID
func (m Model) Id() uuid.UUID {
	return m.id
}

// Name returns the tenant name
func (m Model) Name() string {
	return m.name
}

// Region returns the tenant region
func (m Model) Region() string {
	return m.region
}

// MajorVersion returns the tenant major version
func (m Model) MajorVersion() uint16 {
	return m.majorVersion
}

// MinorVersion returns the tenant minor version
func (m Model) MinorVersion() uint16 {
	return m.minorVersion
}

// String returns a string representation of the tenant
func (m Model) String() string {
	return fmt.Sprintf("Id [%s] Name [%s] Region [%s] Version [%d.%d]", m.Id().String(), m.Name(), m.Region(), m.MajorVersion(), m.MinorVersion())
}

// Builder is used to build a Model
type Builder struct {
	id           uuid.UUID
	name         string
	region       string
	majorVersion uint16
	minorVersion uint16
}

// NewBuilder creates a new Builder
func NewBuilder() *Builder {
	return &Builder{
		id:           uuid.New(),
		name:         "",
		region:       "",
		majorVersion: 0,
		minorVersion: 0,
	}
}

// SetId sets the tenant ID
func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

// SetName sets the tenant name
func (b *Builder) SetName(name string) *Builder {
	b.name = name
	return b
}

// SetRegion sets the tenant region
func (b *Builder) SetRegion(region string) *Builder {
	b.region = region
	return b
}

// SetMajorVersion sets the tenant major version
func (b *Builder) SetMajorVersion(majorVersion uint16) *Builder {
	b.majorVersion = majorVersion
	return b
}

// SetMinorVersion sets the tenant minor version
func (b *Builder) SetMinorVersion(minorVersion uint16) *Builder {
	b.minorVersion = minorVersion
	return b
}

// Build creates a new Model
func (b *Builder) Build() Model {
	return Model{
		id:           b.id,
		name:         b.name,
		region:       b.region,
		majorVersion: b.majorVersion,
		minorVersion: b.minorVersion,
	}
}

// Make converts an Entity to a Model
func Make(e Entity) (Model, error) {
	return NewBuilder().
		SetId(e.ID).
		SetName(e.Name).
		SetRegion(e.Region).
		SetMajorVersion(e.MajorVersion).
		SetMinorVersion(e.MinorVersion).
		Build(), nil
}
