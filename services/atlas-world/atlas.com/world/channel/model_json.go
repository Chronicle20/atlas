package channel

import (
	"encoding/json"
	"time"

	channelConstant "github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

type modelJSON struct {
	Id              uuid.UUID        `json:"id"`
	WorldId         world.Id         `json:"worldId"`
	ChannelId       channelConstant.Id `json:"channelId"`
	IpAddress       string           `json:"ipAddress"`
	Port            int              `json:"port"`
	CurrentCapacity uint32           `json:"currentCapacity"`
	MaxCapacity     uint32           `json:"maxCapacity"`
	CreatedAt       time.Time        `json:"createdAt"`
	ExpRate         float64          `json:"expRate"`
	MesoRate        float64          `json:"mesoRate"`
	ItemDropRate    float64          `json:"itemDropRate"`
	QuestExpRate    float64          `json:"questExpRate"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(&modelJSON{
		Id:              m.id,
		WorldId:         m.worldId,
		ChannelId:       m.channelId,
		IpAddress:       m.ipAddress,
		Port:            m.port,
		CurrentCapacity: m.currentCapacity,
		MaxCapacity:     m.maxCapacity,
		CreatedAt:       m.createdAt,
		ExpRate:         m.expRate,
		MesoRate:        m.mesoRate,
		ItemDropRate:    m.itemDropRate,
		QuestExpRate:    m.questExpRate,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	var aux modelJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.id = aux.Id
	m.worldId = aux.WorldId
	m.channelId = aux.ChannelId
	m.ipAddress = aux.IpAddress
	m.port = aux.Port
	m.currentCapacity = aux.CurrentCapacity
	m.maxCapacity = aux.MaxCapacity
	m.createdAt = aux.CreatedAt
	m.expRate = aux.ExpRate
	m.mesoRate = aux.MesoRate
	m.itemDropRate = aux.ItemDropRate
	m.questExpRate = aux.QuestExpRate
	return nil
}
