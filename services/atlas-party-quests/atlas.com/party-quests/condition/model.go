package condition

type Model struct {
	conditionType string
	operator      string
	value         uint32
	referenceId   uint32
	referenceKey  string
}

func (m Model) Type() string        { return m.conditionType }
func (m Model) Operator() string    { return m.operator }
func (m Model) Value() uint32       { return m.value }
func (m Model) ReferenceId() uint32 { return m.referenceId }
func (m Model) ReferenceKey() string { return m.referenceKey }
