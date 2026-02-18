package condition

import "errors"

type Builder struct {
	conditionType string
	operator      string
	value         uint32
	referenceId   uint32
	referenceKey  string
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

func (b *Builder) SetReferenceKey(key string) *Builder {
	b.referenceKey = key
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
		referenceKey:  b.referenceKey,
	}, nil
}
