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

type modelBuilder struct {
	id           uuid.UUID
	tenantId     uuid.UUID
	resourceName string
	resourceData json.RawMessage
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
		tenantId:     m.tenantId,
		resourceName: m.resourceName,
		resourceData: m.resourceData,
	}
}

// SetID sets the configuration ID
func (b *modelBuilder) SetID(id uuid.UUID) *modelBuilder {
	b.id = id
	return b
}

// SetTenantId sets the tenant ID
func (b *modelBuilder) SetTenantId(tenantId uuid.UUID) *modelBuilder {
	b.tenantId = tenantId
	return b
}

// SetResourceName sets the resource name
func (b *modelBuilder) SetResourceName(resourceName string) *modelBuilder {
	b.resourceName = resourceName
	return b
}

// SetResourceData sets the resource data
func (b *modelBuilder) SetResourceData(resourceData json.RawMessage) *modelBuilder {
	b.resourceData = resourceData
	return b
}

// Build creates a new Model with validation
func (b *modelBuilder) Build() (Model, error) {
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
