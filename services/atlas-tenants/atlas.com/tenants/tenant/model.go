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

// Make converts an Entity to a Model
func Make(e Entity) (Model, error) {
	return NewModelBuilder().
		SetId(e.ID).
		SetName(e.Name).
		SetRegion(e.Region).
		SetMajorVersion(e.MajorVersion).
		SetMinorVersion(e.MinorVersion).
		Build()
}
