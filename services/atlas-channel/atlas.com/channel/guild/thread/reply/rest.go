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

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:        rm.Id,
		posterId:  rm.PosterId,
		message:   rm.Message,
		createdAt: rm.CreatedAt,
	}, nil
}
