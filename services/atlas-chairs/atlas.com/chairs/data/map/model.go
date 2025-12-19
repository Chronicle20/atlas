package _map

type Model struct {
	seats uint32
}

func (m Model) Seats() uint32 {
	return m.seats
}
