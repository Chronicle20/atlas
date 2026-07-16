package visit

// Model is one visit-list entry: a visitor name and their visit count.
type Model struct {
	name  string
	count uint32
}

func (m Model) Name() string  { return m.name }
func (m Model) Count() uint32 { return m.count }
