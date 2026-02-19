package expression

import (
	"encoding/json"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

type Model struct {
	tenant      tenant.Model
	characterId uint32
	field       field.Model
	expression  uint32
	expiration  time.Time
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Tenant      tenant.Model `json:"tenant"`
		CharacterId uint32       `json:"characterId"`
		Field       field.Model  `json:"field"`
		Expression  uint32       `json:"expression"`
		Expiration  time.Time    `json:"expiration"`
	}{
		Tenant:      m.tenant,
		CharacterId: m.characterId,
		Field:       m.field,
		Expression:  m.expression,
		Expiration:  m.expiration,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	t := &struct {
		Tenant      tenant.Model `json:"tenant"`
		CharacterId uint32       `json:"characterId"`
		Field       field.Model  `json:"field"`
		Expression  uint32       `json:"expression"`
		Expiration  time.Time    `json:"expiration"`
	}{}
	if err := json.Unmarshal(data, t); err != nil {
		return err
	}
	m.tenant = t.Tenant
	m.characterId = t.CharacterId
	m.field = t.Field
	m.expression = t.Expression
	m.expiration = t.Expiration
	return nil
}

func (m Model) Expiration() time.Time {
	return m.expiration
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) Expression() uint32 {
	return m.expression
}

func (m Model) Tenant() tenant.Model {
	return m.tenant
}

func (m Model) Field() field.Model {
	return m.field
}

func (m Model) WorldId() world.Id {
	return m.Field().WorldId()
}

func (m Model) ChannelId() channel.Id {
	return m.Field().ChannelId()
}

func (m Model) MapId() _map.Id {
	return m.Field().MapId()
}

func (m Model) Instance() uuid.UUID {
	return m.Field().Instance()
}
