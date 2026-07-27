package tenant

import (
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// RestModel is the JSON:API resource for tenants
type RestModel struct {
	Id           string `json:"-"`
	Name         string `json:"name"`
	Region       string `json:"region"`
	MajorVersion uint16 `json:"majorVersion"`
	MinorVersion uint16 `json:"minorVersion"`
}

// GetID returns the resource ID
func (r RestModel) GetID() string {
	return r.Id
}

// SetID sets the resource ID
func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// GetName returns the resource name
func (r RestModel) GetName() string {
	return "tenants"
}

// Relationship stubs — required because atlas-tenants responses may carry a
// relationships block; without these api2go errors during unmarshal (see
// character/rest.go in this same service for the same pattern).
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func (r *RestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

// Transform converts a Model to a RestModel
func Transform(m tenant.Model) (RestModel, error) {
	return RestModel{
		Id:           m.Id().String(),
		Region:       m.Region(),
		MajorVersion: m.MajorVersion(),
		MinorVersion: m.MinorVersion(),
	}, nil
}

// Extract converts a RestModel to parameters for creating or updating a Model
func Extract(r RestModel) (tenant.Model, error) {
	id, err := uuid.Parse(r.Id)
	if err != nil {
		return tenant.Model{}, err
	}

	return tenant.Register(id, r.Region, r.MajorVersion, r.MinorVersion)
}
