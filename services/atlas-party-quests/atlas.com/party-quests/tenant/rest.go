package tenant

import (
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

type RestModel struct {
	Id           string `json:"-"`
	Name         string `json:"name"`
	Region       string `json:"region"`
	MajorVersion uint16 `json:"majorVersion"`
	MinorVersion uint16 `json:"minorVersion"`
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r RestModel) GetName() string {
	return "tenants"
}

func Extract(r RestModel) (tenant.Model, error) {
	id, err := uuid.Parse(r.Id)
	if err != nil {
		return tenant.Model{}, err
	}

	return tenant.Register(id, r.Region, r.MajorVersion, r.MinorVersion)
}
