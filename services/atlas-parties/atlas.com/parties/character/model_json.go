package character

import (
	"encoding/json"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-constants/job"
	"github.com/google/uuid"
)

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		TenantId uuid.UUID   `json:"tenantId"`
		Id       uint32      `json:"id"`
		Name     string      `json:"name"`
		Level    byte        `json:"level"`
		JobId    job.Id      `json:"jobId"`
		Field    field.Model `json:"field"`
		PartyId  uint32      `json:"partyId"`
		Online   bool        `json:"online"`
		GM       int         `json:"gm"`
	}{m.tenantId, m.id, m.name, m.level, m.jobId, m.field, m.partyId, m.online, m.gm})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var aux struct {
		TenantId uuid.UUID   `json:"tenantId"`
		Id       uint32      `json:"id"`
		Name     string      `json:"name"`
		Level    byte        `json:"level"`
		JobId    job.Id      `json:"jobId"`
		Field    field.Model `json:"field"`
		PartyId  uint32      `json:"partyId"`
		Online   bool        `json:"online"`
		GM       int         `json:"gm"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.tenantId = aux.TenantId
	m.id = aux.Id
	m.name = aux.Name
	m.level = aux.Level
	m.jobId = aux.JobId
	m.field = aux.Field
	m.partyId = aux.PartyId
	m.online = aux.Online
	m.gm = aux.GM
	return nil
}
