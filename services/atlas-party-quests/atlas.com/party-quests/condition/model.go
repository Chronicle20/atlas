package condition

import "errors"

type Model struct {
	conditionType string
	operator      string
	value         uint32
	referenceId   uint32
}

func (m Model) Type() string     { return m.conditionType }
func (m Model) Operator() string { return m.operator }
func (m Model) Value() uint32    { return m.value }
func (m Model) ReferenceId() uint32 { return m.referenceId }

type Builder struct {
	conditionType string
	operator      string
	value         uint32
	referenceId   uint32
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) SetType(t string) *Builder {
	b.conditionType = t
	return b
}

func (b *Builder) SetOperator(op string) *Builder {
	b.operator = op
	return b
}

func (b *Builder) SetValue(v uint32) *Builder {
	b.value = v
	return b
}

func (b *Builder) SetReferenceId(id uint32) *Builder {
	b.referenceId = id
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.conditionType == "" {
		return Model{}, errors.New("type is required")
	}
	if b.operator == "" {
		return Model{}, errors.New("operator is required")
	}
	return Model{
		conditionType: b.conditionType,
		operator:      b.operator,
		value:         b.value,
		referenceId:   b.referenceId,
	}, nil
}
