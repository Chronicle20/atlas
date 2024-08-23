package tenant

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
)

type Model struct {
	id           uuid.UUID
	region       string
	majorVersion uint16
	minorVersion uint16
}

func (m *Model) Id() uuid.UUID {
	return m.id
}

func (m *Model) Region() string {
	return m.region
}

func (m *Model) MajorVersion() uint16 {
	return m.majorVersion
}

func (m *Model) MinorVersion() uint16 {
	return m.minorVersion
}

func (m *Model) MarshalJSON() ([]byte, error) {
	type alias Model
	return json.Marshal(&struct {
		Id           uuid.UUID `json:"id"`
		Region       string    `json:"region"`
		MajorVersion uint16    `json:"majorVersion"`
		MinorVersion uint16    `json:"minorVersion"`
	}{
		Id:           m.id,
		Region:       m.region,
		MajorVersion: m.majorVersion,
		MinorVersion: m.minorVersion,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	t := &struct {
		Id           uuid.UUID `json:"id"`
		Region       string    `json:"region"`
		MajorVersion uint16    `json:"majorVersion"`
		MinorVersion uint16    `json:"minorVersion"`
	}{}

	if err := json.Unmarshal(data, t); err != nil {
		return err
	}

	m.id = t.Id
	m.region = t.Region
	m.majorVersion = t.MajorVersion
	m.minorVersion = t.MinorVersion
	return nil
}

func (m *Model) Is(tenant Model) bool {
	if tenant.Id() != m.Id() {
		return false
	}
	if tenant.Region() != m.Region() {
		return false
	}
	if tenant.MajorVersion() != m.MajorVersion() {
		return false
	}
	if tenant.MinorVersion() != m.MinorVersion() {
		return false
	}
	return true
}

func (m *Model) String() string {
	return fmt.Sprintf("Id [%s] Region [%s] Version [%d.%d]", m.Id().String(), m.Region(), m.MajorVersion(), m.MinorVersion())
}
