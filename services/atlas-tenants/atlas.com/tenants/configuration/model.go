package configuration

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// Model represents a configuration in the domain
type Model struct {
	id           uuid.UUID
	tenantId     uuid.UUID
	resourceName string
	resourceData json.RawMessage
}

// ID returns the configuration ID
func (m Model) ID() uuid.UUID {
	return m.id
}

// TenantId returns the tenant ID
func (m Model) TenantId() uuid.UUID {
	return m.tenantId
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
	return fmt.Sprintf("ID [%s] TenantId [%s] ResourceName [%s]", m.ID().String(), m.TenantId().String(), m.ResourceName())
}

// Make converts an Entity to a Model
func Make(e Entity) (Model, error) {
	return NewModelBuilder().
		SetID(e.ID).
		SetTenantId(e.TenantId).
		SetResourceName(e.ResourceName).
		SetResourceData(e.ResourceData).
		Build()
}
