package reactor

import (
	"atlas-data/point"
	"github.com/Chronicle20/atlas-model/model"
	"strconv"
)

type RestModel struct {
	Id          string                           `json:"-"`
	TL          point.RestModel                  `json:"tl"`
	BR          point.RestModel                  `json:"br"`
	StateInfo   map[int8][]ReactorStateRestModel `json:"stateInfo"`
	TimeoutInfo map[int8]int32                   `json:"timeoutInfo"`
}

func (r RestModel) GetName() string {
	return "reactors"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func Transform(m Model) (RestModel, error) {
	tl, err := model.Map(point.Transform)(model.FixedProvider(m.TL))()
	if err != nil {
		return RestModel{}, err
	}
	br, err := model.Map(point.Transform)(model.FixedProvider(m.BR))()
	if err != nil {
		return RestModel{}, err
	}
	si, err := model.Map(TransformStateInfo)(model.FixedProvider(m.StateInfo))()

	return RestModel{
		Id:          strconv.Itoa(int(m.Id)),
		TL:          tl,
		BR:          br,
		StateInfo:   si,
		TimeoutInfo: m.TimeoutInfo,
	}, nil
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
	si, err := model.Map(ExtractStateInfo)(model.FixedProvider(rm.StateInfo))()

	id, err := strconv.Atoi(rm.Id)
	if err != nil {
		return Model{}, err
	}

	return Model{
		Id:          uint32(id),
		TL:          tl,
		BR:          br,
		StateInfo:   si,
		TimeoutInfo: rm.TimeoutInfo,
	}, nil
}

type ReactorStateRestModel struct {
	Type         int32                 `json:"type"`
	ReactorItem  *ReactorItemRestModel `json:"reactorItem"`
	ActiveSkills []uint32              `json:"activeSkills"`
	NextState    int8                  `json:"nextState"`
}

func TransformStateInfo(m map[int8][]ReactorState) (map[int8][]ReactorStateRestModel, error) {
	res := make(map[int8][]ReactorStateRestModel)
	for k, vs := range m {
		res[k] = make([]ReactorStateRestModel, 0)
		for _, v := range vs {
			var ri *ReactorItemRestModel
			if v.ReactorItem != nil {
				rm, err := TransformReactorItem(*v.ReactorItem)
				if err != nil {
					return nil, err
				}
				ri = &rm
			}
			rsrm := ReactorStateRestModel{
				Type:         v.Type,
				ActiveSkills: v.ActiveSkills,
				NextState:    v.NextState,
			}
			if ri != nil {
				rsrm.ReactorItem = ri
			}

			res[k] = append(res[k], rsrm)
		}
	}
	return res, nil
}

func ExtractStateInfo(rm map[int8][]ReactorStateRestModel) (map[int8][]ReactorState, error) {
	res := make(map[int8][]ReactorState)
	for k, vs := range rm {
		res[k] = make([]ReactorState, 0)
		for _, v := range vs {
			var ri *ReactorItem
			if v.ReactorItem != nil {
				r, err := ExtractReactorItem(*v.ReactorItem)
				if err != nil {
					return nil, err
				}
				ri = &r
			}
			rsr := ReactorState{
				Type:         v.Type,
				ActiveSkills: v.ActiveSkills,
				NextState:    v.NextState,
			}
			if ri != nil {
				rsr.ReactorItem = ri
			}
			res[k] = append(res[k], rsr)
		}
	}
	return res, nil
}

type ReactorItemRestModel struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint16 `json:"quantity"`
}

func TransformReactorItem(m ReactorItem) (ReactorItemRestModel, error) {
	return ReactorItemRestModel{
		ItemId:   m.ItemId,
		Quantity: m.Quantity,
	}, nil
}

func ExtractReactorItem(rm ReactorItemRestModel) (ReactorItem, error) {
	return ReactorItem{
		ItemId:   rm.ItemId,
		Quantity: rm.Quantity,
	}, nil
}
