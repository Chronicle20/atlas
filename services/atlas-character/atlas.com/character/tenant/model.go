package tenant

import (
	"fmt"
	"github.com/google/uuid"
)

type Model struct {
	Id           uuid.UUID `json:"id"`
	Region       string    `json:"region"`
	MajorVersion uint16    `json:"majorVersion"`
	MinorVersion uint16    `json:"minorVersion"`
}

func New(id uuid.UUID, region string, majorVersion uint16, minorVersion uint16) Model {
	return Model{
		Id:           id,
		Region:       region,
		MajorVersion: majorVersion,
		MinorVersion: minorVersion,
	}
}

func (m Model) String() string {
	return fmt.Sprintf("Id [%s] Region [%s] Version [%d.%d]", m.Id.String(), m.Region, m.MajorVersion, m.MinorVersion)
}

func (m Model) Is(tenant Model) bool {
	if tenant.Id != m.Id {
		return false
	}
	if tenant.Region != m.Region {
		return false
	}
	if tenant.MajorVersion != m.MajorVersion {
		return false
	}
	if tenant.MinorVersion != m.MinorVersion {
		return false
	}
	return true
}
