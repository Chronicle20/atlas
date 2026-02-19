package configuration

import (
	"encoding/json"

	"github.com/google/uuid"
)

type entityBuilder struct {
	id           uuid.UUID
	tenantId     uuid.UUID
	resourceName string
	resourceData json.RawMessage
}

// NewEntityBuilder creates a new entity builder
func NewEntityBuilder() *entityBuilder {
	return &entityBuilder{}
}

// SetID sets the configuration ID
func (b *entityBuilder) SetID(id uuid.UUID) *entityBuilder {
	b.id = id
	return b
}

// SetTenantId sets the tenant ID
func (b *entityBuilder) SetTenantId(tenantId uuid.UUID) *entityBuilder {
	b.tenantId = tenantId
	return b
}

// SetResourceName sets the resource name
func (b *entityBuilder) SetResourceName(resourceName string) *entityBuilder {
	b.resourceName = resourceName
	return b
}

// SetResourceData sets the resource data
func (b *entityBuilder) SetResourceData(resourceData json.RawMessage) *entityBuilder {
	b.resourceData = resourceData
	return b
}

// Build creates a new Entity
func (b *entityBuilder) Build() Entity {
	return Entity{
		ID:           b.id,
		TenantId:     b.tenantId,
		ResourceName: b.resourceName,
		ResourceData: b.resourceData,
	}
}

// FromModel creates an Entity from a Model
func FromModel(m Model) Entity {
	return NewEntityBuilder().
		SetID(m.ID()).
		SetTenantId(m.TenantId()).
		SetResourceName(m.ResourceName()).
		SetResourceData(m.ResourceData()).
		Build()
}
