package point

type Model struct {
	x int16
	y int16
}

func (p Model) X() int16 {
	return p.x
}

func (p Model) Y() int16 {
	return p.y
}
