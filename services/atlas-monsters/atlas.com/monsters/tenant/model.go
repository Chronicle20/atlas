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

func (m Model) String() string {
	return fmt.Sprintf("Id [%s] Region [%s] Version [%d.%d]", m.Id.String(), m.Region, m.MajorVersion, m.MinorVersion)
}

func New(id uuid.UUID, region string, majorVersion uint16, minorVersion uint16) Model {
	return Model{
		Id:           id,
		Region:       region,
		MajorVersion: majorVersion,
		MinorVersion: minorVersion,
	}
}
