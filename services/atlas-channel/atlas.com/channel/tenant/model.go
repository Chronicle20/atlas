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
