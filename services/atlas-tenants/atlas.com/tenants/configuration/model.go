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

// Make converts an Entity to a Model
func Make(e Entity) (Model, error) {
	return NewModelBuilder().
		SetID(e.ID).
		SetTenantID(e.TenantID).
		SetResourceName(e.ResourceName).
		SetResourceData(e.ResourceData).
		Build()
}
