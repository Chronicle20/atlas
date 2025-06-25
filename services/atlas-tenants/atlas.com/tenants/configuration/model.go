package configuration

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
)

// Model represents a configuration in the domain
type Model struct {
	tenantID     uuid.UUID
	resourceName string
	resourceData json.RawMessage
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
	return fmt.Sprintf("TenantID [%s] ResourceName [%s]", m.TenantID().String(), m.ResourceName())
}

// Builder is used to build a Model
type Builder struct {
	tenantID     uuid.UUID
	resourceName string
	resourceData json.RawMessage
}

// NewBuilder creates a new Builder
func NewBuilder() *Builder {
	return &Builder{
		tenantID:     uuid.Nil,
		resourceName: "",
		resourceData: nil,
	}
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
		tenantID:     b.tenantID,
		resourceName: b.resourceName,
		resourceData: b.resourceData,
	}
}

// Make converts an Entity to a Model
func Make(e Entity) (Model, error) {
	return NewBuilder().
		SetTenantID(e.TenantID).
		SetResourceName(e.ResourceName).
		SetResourceData(e.ResourceData).
		Build(), nil
}