package tenant

import "github.com/google/uuid"

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

func (m Model) Equals(o Model) bool {
	if m.Id != o.Id {
		return false
	}
	if m.Region != o.Region {
		return false
	}
	if m.MajorVersion != o.MajorVersion {
		return false
	}
	if m.MinorVersion != o.MinorVersion {
		return false
	}
	return true
}
