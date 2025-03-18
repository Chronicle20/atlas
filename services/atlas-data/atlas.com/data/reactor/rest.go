package reactor

import (
	"atlas-data/point"
	"strconv"
)

type RestModel struct {
	Id          uint32                           `json:"-"`
	TL          point.RestModel                  `json:"tl"`
	BR          point.RestModel                  `json:"br"`
	StateInfo   map[int8][]ReactorStateRestModel `json:"stateInfo"`
	TimeoutInfo map[int8]int32                   `json:"timeoutInfo"`
}

func (r RestModel) GetName() string {
	return "reactors"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r RestModel) GetId() uint32 {
	return r.Id
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

type ReactorStateRestModel struct {
	Type         int32                 `json:"type"`
	ReactorItem  *ReactorItemRestModel `json:"reactorItem"`
	ActiveSkills []uint32              `json:"activeSkills"`
	NextState    int8                  `json:"nextState"`
}

type ReactorItemRestModel struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint16 `json:"quantity"`
}
