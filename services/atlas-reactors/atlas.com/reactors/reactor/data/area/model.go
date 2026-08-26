package area

import "atlas-reactors/reactor/data/point"

type Model struct {
	tl point.Model
	br point.Model
}

func (m Model) TL() point.Model {
	return m.tl
}

func (m Model) BR() point.Model {
	return m.br
}
