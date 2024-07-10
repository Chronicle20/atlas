package world

import "strconv"

type RestModel struct {
	Id                 string `json:"-"`
	Name               string `json:"name"`
	Flag               int    `json:"flag"`
	Message            string `json:"message"`
	EventMessage       string `json:"eventMessage"`
	Recommended        bool   `json:"recommended"`
	RecommendedMessage string `json:"recommendedMessage"`
	CapacityStatus     uint32 `json:"capacityStatus"`
}

func (r RestModel) GetName() string {
	return "worlds"
}

func (r RestModel) GetID() string {
	return r.Id
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:                 strconv.Itoa(int(m.id)),
		Name:               m.name,
		Flag:               getFlag(m.flag),
		Message:            m.message,
		EventMessage:       m.eventMessage,
		Recommended:        m.recommendedMessage != "",
		RecommendedMessage: m.recommendedMessage,
		CapacityStatus:     m.capacityStatus,
	}, nil
}
