package reactor

import (
	"github.com/Chronicle20/atlas-model/model"
	"strconv"
)

type RestModel struct {
	Id          string                           `json:"-"`
	TL          PointRestModel                   `json:"tl"`
	BR          PointRestModel                   `json:"br"`
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
	tl, err := model.Map(TransformPoint)(model.FixedProvider(m.tl))()
	if err != nil {
		return RestModel{}, err
	}
	br, err := model.Map(TransformPoint)(model.FixedProvider(m.br))()
	if err != nil {
		return RestModel{}, err
	}
	si, err := model.Map(TransformStateInfo)(model.FixedProvider(m.stateInfo))()

	return RestModel{
		Id:          strconv.Itoa(int(m.id)),
		TL:          tl,
		BR:          br,
		StateInfo:   si,
		TimeoutInfo: m.timeoutInfo,
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
			if v.reactorItem != nil {
				rm, err := TransformReactorItem(*v.reactorItem)
				if err != nil {
					return nil, err
				}
				ri = &rm
			}
			rsrm := ReactorStateRestModel{
				Type:         v.Type(),
				ActiveSkills: v.ActiveSkills(),
				NextState:    v.NextState(),
			}
			if ri != nil {
				rsrm.ReactorItem = ri
			}

			res[k] = append(res[k], rsrm)
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
		ItemId:   m.itemId,
		Quantity: m.quantity,
	}, nil
}

type PointRestModel struct {
	X int16 `json:"x"`
	Y int16 `json:"y"`
}

func TransformPoint(m Point) (PointRestModel, error) {
	return PointRestModel{
		X: m.X(),
		Y: m.Y(),
	}, nil
}
