package party

import (
	"encoding/json"

	"github.com/google/uuid"
)

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		TenantId uuid.UUID `json:"tenantId"`
		Id       uint32    `json:"id"`
		LeaderId uint32    `json:"leaderId"`
		Members  []uint32  `json:"members"`
	}{m.tenantId, m.id, m.leaderId, m.members})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var aux struct {
		TenantId uuid.UUID `json:"tenantId"`
		Id       uint32    `json:"id"`
		LeaderId uint32    `json:"leaderId"`
		Members  []uint32  `json:"members"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.tenantId = aux.TenantId
	m.id = aux.Id
	m.leaderId = aux.LeaderId
	m.members = aux.Members
	return nil
}
