package reactor

import (
	"atlas-data/point"
	"strconv"
)

type RestModel struct {
	Id                   uint32                           `json:"-"`
	Name                 string                           `json:"name"`
	TL                   point.RestModel                  `json:"tl"`
	BR                   point.RestModel                  `json:"br"`
	ActivateByTouch      bool                             `json:"activateByTouch"`
	TouchAreaInfo        map[int8]AreaRestModel           `json:"touchAreaInfo"`
	StateInfo            map[int8][]ReactorStateRestModel `json:"stateInfo"`
	TimeoutInfo          map[int8]int32                   `json:"timeoutInfo"`
	TimeoutNextStateInfo map[int8]int8                    `json:"timeoutNextStateInfo"`
}

type AreaRestModel struct {
	TL point.RestModel `json:"tl"`
	BR point.RestModel `json:"br"`
}

func (r RestModel) GetName() string {
	return "reactors"
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
