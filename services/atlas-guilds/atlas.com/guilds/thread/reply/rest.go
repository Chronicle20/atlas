package reply

import (
	"strconv"
	"time"
)

type RestModel struct {
	Id        uint32    `json:"id"`
	PosterId  uint32    `json:"posterId"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}

func (r RestModel) GetName() string {
	return "replies"
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

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:        m.id,
		PosterId:  m.posterId,
		Message:   m.message,
		CreatedAt: m.createdAt,
	}, nil
}
