package reactor

type Point struct {
	x int16
	y int16
}

func (p Point) X() int16 {
	return p.x
}

func (p Point) Y() int16 {
	return p.y
}

type Model struct {
	id          uint32
	tl          Point
	br          Point
	stateInfo   map[int8][]ReactorState
	timeoutInfo map[int8]int32
}

func NewModel(id uint32) Model {
	return Model{
		id:          id,
		tl:          Point{0, 0},
		br:          Point{0, 0},
		stateInfo:   make(map[int8][]ReactorState),
		timeoutInfo: make(map[int8]int32),
	}
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) AddState(state int8, data []ReactorState, timeout int32) Model {
	nm := Model{
		id:          m.id,
		tl:          m.tl,
		br:          m.br,
		stateInfo:   m.stateInfo,
		timeoutInfo: m.timeoutInfo,
	}

	nm.stateInfo[state] = data
	if timeout > -1 {
		nm.timeoutInfo[state] = timeout
	}
	return nm
}

func (m Model) SetTL(x int32, y int32) Model {
	nm := Model{
		id:          m.id,
		tl:          Point{int16(x), int16(y)},
		br:          m.br,
		stateInfo:   m.stateInfo,
		timeoutInfo: m.timeoutInfo,
	}
	return nm
}

func (m Model) SetRB(x int32, y int32) Model {
	nm := Model{
		id:          m.id,
		tl:          m.tl,
		br:          Point{int16(x), int16(y)},
		stateInfo:   m.stateInfo,
		timeoutInfo: m.timeoutInfo,
	}
	return nm
}

type ReactorState struct {
	theType      int32
	reactorItem  *ReactorItem
	activeSkills []uint32
	nextState    int8
}

func (s ReactorState) Type() int32 {
	return s.theType
}

func (s ReactorState) NextState() int8 {
	return s.nextState
}

func (s ReactorState) ActiveSkills() []uint32 {
	return s.activeSkills
}

type ReactorItem struct {
	itemId   uint32
	quantity uint16
}
