package tenant

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrNameRequired   = errors.New("tenant name is required")
	ErrRegionRequired = errors.New("tenant region is required")
)

type builder struct {
	id           uuid.UUID
	name         string
	region       string
	majorVersion uint16
	minorVersion uint16
	environment  string
}

// NewBuilder creates a new model builder
func NewBuilder() *builder {
	return &builder{
		id: uuid.New(),
	}
}

// CloneModel creates a builder from an existing model
func CloneModel(m Model) *builder {
	return &builder{
		id:           m.id,
		name:         m.name,
		region:       m.region,
		majorVersion: m.majorVersion,
		minorVersion: m.minorVersion,
		environment:  m.environment,
	}
}

// SetId sets the tenant ID
func (b *builder) SetId(id uuid.UUID) *builder {
	b.id = id
	return b
}

// SetName sets the tenant name
func (b *builder) SetName(name string) *builder {
	b.name = name
	return b
}

// SetRegion sets the tenant region
func (b *builder) SetRegion(region string) *builder {
	b.region = region
	return b
}

// SetMajorVersion sets the tenant major version
func (b *builder) SetMajorVersion(majorVersion uint16) *builder {
	b.majorVersion = majorVersion
	return b
}

// SetMinorVersion sets the tenant minor version
func (b *builder) SetMinorVersion(minorVersion uint16) *builder {
	b.minorVersion = minorVersion
	return b
}

// SetEnvironment sets the tenant environment
func (b *builder) SetEnvironment(environment string) *builder {
	b.environment = environment
	return b
}

// Build creates a new Model with validation
func (b *builder) Build() (Model, error) {
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
		environment:  b.environment,
	}, nil
}
