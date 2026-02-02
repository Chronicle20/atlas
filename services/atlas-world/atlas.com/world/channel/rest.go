package channel

import (
	"github.com/google/uuid"
	"time"
)

type RestModel struct {
	Id              uuid.UUID `json:"-"`
	WorldId         byte      `json:"worldId"`
	ChannelId       byte      `json:"channelId"`
	IpAddress       string    `json:"ipAddress"`
	Port            int       `json:"port"`
	CurrentCapacity uint32    `json:"currentCapacity"`
	MaxCapacity     uint32    `json:"maxCapacity"`
	CreatedAt       time.Time `json:"createdAt"`
	ExpRate         float64   `json:"expRate"`
	MesoRate        float64   `json:"mesoRate"`
	ItemDropRate    float64   `json:"itemDropRate"`
	QuestExpRate    float64   `json:"questExpRate"`
}

func (r RestModel) GetName() string {
	return "channels"
}

func (r RestModel) GetID() string {
	return r.Id.String()
}

func (r *RestModel) SetID(id string) error {
	r.Id = uuid.MustParse(id)
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:              m.Id(),
		WorldId:         m.WorldId(),
		ChannelId:       m.ChannelId(),
		IpAddress:       m.IpAddress(),
		Port:            m.Port(),
		CurrentCapacity: m.CurrentCapacity(),
		MaxCapacity:     m.MaxCapacity(),
		CreatedAt:       m.CreatedAt(),
		ExpRate:         m.ExpRate(),
		MesoRate:        m.MesoRate(),
		ItemDropRate:    m.ItemDropRate(),
		QuestExpRate:    m.QuestExpRate(),
	}, nil
}

// Extract converts a RestModel to a Model using the Builder pattern
func Extract(r RestModel) (Model, error) {
	return NewModelBuilder().
		SetId(r.Id).
		SetWorldId(r.WorldId).
		SetChannelId(r.ChannelId).
		SetIpAddress(r.IpAddress).
		SetPort(r.Port).
		SetCurrentCapacity(r.CurrentCapacity).
		SetMaxCapacity(r.MaxCapacity).
		SetCreatedAt(r.CreatedAt).
		SetExpRate(r.ExpRate).
		SetMesoRate(r.MesoRate).
		SetItemDropRate(r.ItemDropRate).
		SetQuestExpRate(r.QuestExpRate).
		Build()
}
