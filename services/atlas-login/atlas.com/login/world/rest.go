package world

import "strconv"

type RestModel struct {
	Id                 uint32 `json:"-"`
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
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Extract(m RestModel) (Model, error) {
	return Model{
		id:                 byte(m.Id),
		name:               m.Name,
		state:              State(m.Flag),
		message:            m.Message,
		eventMessage:       m.EventMessage,
		recommended:        m.Recommended,
		recommendedMessage: m.RecommendedMessage,
		capacityStatus:     Status(m.CapacityStatus),
		channelLoad:        nil,
	}, nil
}
