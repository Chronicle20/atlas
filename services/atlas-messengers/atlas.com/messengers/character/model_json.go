package character

import (
	"encoding/json"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

type modelJSON struct {
	TenantId    uuid.UUID  `json:"tenantId"`
	Id          uint32     `json:"id"`
	Name        string     `json:"name"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	MessengerId uint32     `json:"messengerId"`
	Online      bool       `json:"online"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(&modelJSON{
		TenantId:    m.tenantId,
		Id:          m.id,
		Name:        m.name,
		WorldId:     m.ch.WorldId(),
		ChannelId:   m.ch.Id(),
		MessengerId: m.messengerId,
		Online:      m.online,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var aux modelJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.tenantId = aux.TenantId
	m.id = aux.Id
	m.name = aux.Name
	m.ch = channel.NewModel(aux.WorldId, aux.ChannelId)
	m.messengerId = aux.MessengerId
	m.online = aux.Online
	return nil
}
