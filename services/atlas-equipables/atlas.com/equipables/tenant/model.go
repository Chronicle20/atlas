package tenant

import "github.com/google/uuid"

type Model struct {
	id           uuid.UUID
	region       string
	majorVersion uint16
	minorVersion uint16
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) Region() string {
	return m.region
}

func (m Model) MajorVersion() uint16 {
	return m.majorVersion
}

func (m Model) MinorVersion() uint16 {
	return m.minorVersion
}

func New(id uuid.UUID, region string, majorVersion uint16, minorVersion uint16) Model {
	return Model{
		id:           id,
		region:       region,
		majorVersion: majorVersion,
		minorVersion: minorVersion,
	}
}
