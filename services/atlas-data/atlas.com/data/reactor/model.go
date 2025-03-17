package reactor

import "atlas-data/point"

type Model struct {
	Id          uint32                  `json:"id"`
	TL          point.Model             `json:"tl"`
	BR          point.Model             `json:"br"`
	StateInfo   map[int8][]ReactorState `json:"state_info"`
	TimeoutInfo map[int8]int32          `json:"timeout_info"`
}

func NewModel(id uint32) Model {
	return Model{
		Id:          id,
		TL:          point.Model{},
		BR:          point.Model{},
		StateInfo:   make(map[int8][]ReactorState),
		TimeoutInfo: make(map[int8]int32),
	}
}

func (m Model) GetId() uint32 {
	return m.Id
}

func (m Model) AddState(state int8, data []ReactorState, timeout int32) Model {
	nm := Model{
		Id:          m.Id,
		TL:          m.TL,
		BR:          m.BR,
		StateInfo:   m.StateInfo,
		TimeoutInfo: m.TimeoutInfo,
	}

	nm.StateInfo[state] = data
	if timeout > -1 {
		nm.TimeoutInfo[state] = timeout
	}
	return nm
}

func (m Model) SetTL(x int32, y int32) Model {
	nm := Model{
		Id:          m.Id,
		TL:          point.Model{X: int16(x), Y: int16(y)},
		BR:          m.BR,
		StateInfo:   m.StateInfo,
		TimeoutInfo: m.TimeoutInfo,
	}
	return nm
}

func (m Model) SetRB(x int32, y int32) Model {
	nm := Model{
		Id:          m.Id,
		TL:          m.TL,
		BR:          point.Model{X: int16(x), Y: int16(y)},
		StateInfo:   m.StateInfo,
		TimeoutInfo: m.TimeoutInfo,
	}
	return nm
}

type ReactorState struct {
	Type         int32        `json:"type"`
	ReactorItem  *ReactorItem `json:"reactor_item,omitempty"`
	ActiveSkills []uint32     `json:"active_skills"`
	NextState    int8         `json:"next_state"`
}

type ReactorItem struct {
	ItemId   uint32 `json:"item_id"`
	Quantity uint16 `json:"quantity"`
}
