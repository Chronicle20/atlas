package configuration

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
)

// Model represents a configuration in the domain
type Model struct {
	id           uuid.UUID
	tenantID     uuid.UUID
	resourceName string
	resourceData json.RawMessage
}

// ID returns the configuration ID
func (m Model) ID() uuid.UUID {
	return m.id
}

// TenantID returns the tenant ID
func (m Model) TenantID() uuid.UUID {
	return m.tenantID
}

// ResourceName returns the resource name
func (m Model) ResourceName() string {
	return m.resourceName
}

// ResourceData returns the resource data
func (m Model) ResourceData() json.RawMessage {
	return m.resourceData
}

// String returns a string representation of the configuration
func (m Model) String() string {
	return fmt.Sprintf("ID [%s] TenantID [%s] ResourceName [%s]", m.ID().String(), m.TenantID().String(), m.ResourceName())
}

// Builder is used to build a Model
type Builder struct {
	id           uuid.UUID
	tenantID     uuid.UUID
	resourceName string
	resourceData json.RawMessage
}

// NewBuilder creates a new Builder
func NewBuilder() *Builder {
	return &Builder{
		id:           uuid.New(),
		tenantID:     uuid.Nil,
		resourceName: "",
		resourceData: nil,
	}
}

// SetID sets the configuration ID
func (b *Builder) SetID(id uuid.UUID) *Builder {
	b.id = id
	return b
}

// SetTenantID sets the tenant ID
func (b *Builder) SetTenantID(tenantID uuid.UUID) *Builder {
	b.tenantID = tenantID
	return b
}

// SetResourceName sets the resource name
func (b *Builder) SetResourceName(resourceName string) *Builder {
	b.resourceName = resourceName
	return b
}

// SetResourceData sets the resource data
func (b *Builder) SetResourceData(resourceData json.RawMessage) *Builder {
	b.resourceData = resourceData
	return b
}

// Build creates a new Model
func (b *Builder) Build() Model {
	return Model{
		id:           b.id,
		tenantID:     b.tenantID,
		resourceName: b.resourceName,
		resourceData: b.resourceData,
	}
}

// Make converts an Entity to a Model
func Make(e Entity) (Model, error) {
	return NewBuilder().
		SetID(e.ID).
		SetTenantID(e.TenantID).
		SetResourceName(e.ResourceName).
		SetResourceData(e.ResourceData).
		Build(), nil
}
