package coordinator

import (
	"encoding/json"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
)

type modelJSON struct {
	Tenant    tenant.Model    `json:"tenant"`
	WorldId   world.Id        `json:"worldId"`
	ChannelId channel.Id      `json:"channelId"`
	LeaderId  uint32          `json:"leaderId"`
	Name      string          `json:"name"`
	Requests  []uint32        `json:"requests"`
	Responses map[uint32]bool `json:"responses"`
	Age       time.Time       `json:"age"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(&modelJSON{
		Tenant:    m.tenant,
		WorldId:   m.channel.WorldId(),
		ChannelId: m.channel.Id(),
		LeaderId:  m.leaderId,
		Name:      m.name,
		Requests:  m.requests,
		Responses: m.responses,
		Age:       m.age,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var aux modelJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.tenant = aux.Tenant
	m.channel = channel.NewModel(aux.WorldId, aux.ChannelId)
	m.leaderId = aux.LeaderId
	m.name = aux.Name
	m.requests = aux.Requests
	m.responses = aux.Responses
	m.age = aux.Age
	return nil
}
