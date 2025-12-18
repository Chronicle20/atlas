package key

type Model struct {
	characterId uint32
	key         int32
	theType     int8
	action      int32
}

func (m Model) Key() int32 {
	return m.key
}

func (m Model) Type() int8 {
	return m.theType
}

func (m Model) Action() int32 {
	return m.action
}
