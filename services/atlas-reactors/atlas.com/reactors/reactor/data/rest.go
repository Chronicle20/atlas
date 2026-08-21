package data

import (
	"atlas-reactors/reactor/data/area"
	"atlas-reactors/reactor/data/point"
	"atlas-reactors/reactor/data/state"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type RestModel struct {
	Id                   uint32                     `json:"-"`
	Name                 string                     `json:"name"`
	TL                   point.RestModel            `json:"tl"`
	BR                   point.RestModel            `json:"br"`
	ActivateByTouch      bool                       `json:"activateByTouch"`
	TouchAreaInfo        map[int8]area.RestModel    `json:"touchAreaInfo"`
	StateInfo            map[int8][]state.RestModel `json:"stateInfo"`
	TimeoutInfo          map[int8]int32             `json:"timeoutInfo"`
	TimeoutNextStateInfo map[int8]int8              `json:"timeoutNextStateInfo"`
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

func Extract(rm RestModel) (Model, error) {
	tl, err := model.Map(point.Extract)(model.FixedProvider(rm.TL))()
	if err != nil {
		return Model{}, err
	}
	br, err := model.Map(point.Extract)(model.FixedProvider(rm.BR))()
	if err != nil {
		return Model{}, err
	}
	si := make(map[int8][]state.Model)
	for k, vs := range rm.StateInfo {
		si[k] = make([]state.Model, 0)
		for _, v := range vs {
			sm, err := state.Extract(v)
			if err != nil {
				return Model{}, err
			}

			si[k] = append(si[k], sm)
		}
	}

	var tai map[int8]area.Model
	if rm.TouchAreaInfo != nil {
		tai = make(map[int8]area.Model)
		for k, v := range rm.TouchAreaInfo {
			am, err := area.Extract(v)
			if err != nil {
				return Model{}, err
			}
			tai[k] = am
		}
	}

	return Model{
		name:                 rm.Name,
		tl:                   tl,
		br:                   br,
		activateByTouch:      rm.ActivateByTouch,
		touchAreaInfo:        tai,
		stateInfo:            si,
		timeoutInfo:          rm.TimeoutInfo,
		timeoutNextStateInfo: rm.TimeoutNextStateInfo,
	}, nil
}
