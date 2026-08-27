package configuration

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrTenantIdRequired     = errors.New("tenant id is required")
	ErrResourceNameRequired = errors.New("resource name is required")
)

type builder struct {
	id           uuid.UUID
	tenantId     uuid.UUID
	resourceName string
	resourceData json.RawMessage
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
		tenantId:     m.tenantId,
		resourceName: m.resourceName,
		resourceData: m.resourceData,
	}
}

// SetID sets the configuration ID
func (b *builder) SetID(id uuid.UUID) *builder {
	b.id = id
	return b
}

// SetTenantId sets the tenant ID
func (b *builder) SetTenantId(tenantId uuid.UUID) *builder {
	b.tenantId = tenantId
	return b
}

// SetResourceName sets the resource name
func (b *builder) SetResourceName(resourceName string) *builder {
	b.resourceName = resourceName
	return b
}

// SetResourceData sets the resource data
func (b *builder) SetResourceData(resourceData json.RawMessage) *builder {
	b.resourceData = resourceData
	return b
}

// Build creates a new Model with validation
func (b *builder) Build() (Model, error) {
	if b.tenantId == uuid.Nil {
		return Model{}, ErrTenantIdRequired
	}
	if b.resourceName == "" {
		return Model{}, ErrResourceNameRequired
	}
	return Model{
		id:           b.id,
		tenantId:     b.tenantId,
		resourceName: b.resourceName,
		resourceData: b.resourceData,
	}, nil
}
