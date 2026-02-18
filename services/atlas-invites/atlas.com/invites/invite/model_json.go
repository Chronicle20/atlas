package invite

import (
	"encoding/json"
	"time"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
)

type modelJSON struct {
	Tenant       tenant.Model `json:"tenant"`
	Id           uint32       `json:"id"`
	InviteType   string       `json:"inviteType"`
	ReferenceId  uint32       `json:"referenceId"`
	OriginatorId uint32       `json:"originatorId"`
	TargetId     uint32       `json:"targetId"`
	WorldId      world.Id     `json:"worldId"`
	Age          time.Time    `json:"age"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(&modelJSON{
		Tenant:       m.tenant,
		Id:           m.id,
		InviteType:   m.inviteType,
		ReferenceId:  m.referenceId,
		OriginatorId: m.originatorId,
		TargetId:     m.targetId,
		WorldId:      m.worldId,
		Age:          m.age,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var aux modelJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.tenant = aux.Tenant
	m.id = aux.Id
	m.inviteType = aux.InviteType
	m.referenceId = aux.ReferenceId
	m.originatorId = aux.OriginatorId
	m.targetId = aux.TargetId
	m.worldId = aux.WorldId
	m.age = aux.Age
	return nil
}
