package tenant

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrNameRequired   = errors.New("tenant name is required")
	ErrRegionRequired = errors.New("tenant region is required")
)

type modelBuilder struct {
	id           uuid.UUID
	name         string
	region       string
	majorVersion uint16
	minorVersion uint16
}

// NewModelBuilder creates a new model builder
func NewModelBuilder() *modelBuilder {
	return &modelBuilder{
		id: uuid.New(),
	}
}

// CloneModel creates a builder from an existing model
func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
		id:           m.id,
		name:         m.name,
		region:       m.region,
		majorVersion: m.majorVersion,
		minorVersion: m.minorVersion,
	}
}

// SetId sets the tenant ID
func (b *modelBuilder) SetId(id uuid.UUID) *modelBuilder {
	b.id = id
	return b
}

// SetName sets the tenant name
func (b *modelBuilder) SetName(name string) *modelBuilder {
	b.name = name
	return b
}

// SetRegion sets the tenant region
func (b *modelBuilder) SetRegion(region string) *modelBuilder {
	b.region = region
	return b
}

// SetMajorVersion sets the tenant major version
func (b *modelBuilder) SetMajorVersion(majorVersion uint16) *modelBuilder {
	b.majorVersion = majorVersion
	return b
}

// SetMinorVersion sets the tenant minor version
func (b *modelBuilder) SetMinorVersion(minorVersion uint16) *modelBuilder {
	b.minorVersion = minorVersion
	return b
}

// Build creates a new Model with validation
func (b *modelBuilder) Build() (Model, error) {
	if b.name == "" {
		return Model{}, ErrNameRequired
	}
	if b.region == "" {
		return Model{}, ErrRegionRequired
	}
	return Model{
		id:           b.id,
		name:         b.name,
		region:       b.region,
		majorVersion: b.majorVersion,
		minorVersion: b.minorVersion,
	}, nil
}
