package messenger

import (
	"encoding/json"

	"github.com/google/uuid"
)

type memberModelJSON struct {
	Id   uint32 `json:"id"`
	Slot byte   `json:"slot"`
}

type modelJSON struct {
	TenantId uuid.UUID         `json:"tenantId"`
	Id       uint32            `json:"id"`
	Members  []memberModelJSON `json:"members"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	members := make([]memberModelJSON, len(m.members))
	for i, mm := range m.members {
		members[i] = memberModelJSON{Id: mm.id, Slot: mm.slot}
	}
	return json.Marshal(&modelJSON{
		TenantId: m.tenantId,
		Id:       m.id,
		Members:  members,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var aux modelJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.tenantId = aux.TenantId
	m.id = aux.Id
	m.members = make([]MemberModel, len(aux.Members))
	for i, mm := range aux.Members {
		m.members[i] = MemberModel{id: mm.Id, slot: mm.Slot}
	}
	return nil
}
