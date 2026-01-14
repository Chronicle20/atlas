package stackable

import "errors"

// ModelBuilder for constructing Model instances
type ModelBuilder struct {
	assetId  uint32
	quantity uint32
	ownerId  uint32
	flag     uint16
}

func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{
		quantity: 1,
	}
}

func (b *ModelBuilder) SetAssetId(assetId uint32) *ModelBuilder {
	b.assetId = assetId
	return b
}

func (b *ModelBuilder) SetQuantity(quantity uint32) *ModelBuilder {
	b.quantity = quantity
	return b
}

func (b *ModelBuilder) SetOwnerId(ownerId uint32) *ModelBuilder {
	b.ownerId = ownerId
	return b
}

func (b *ModelBuilder) SetFlag(flag uint16) *ModelBuilder {
	b.flag = flag
	return b
}

func (b *ModelBuilder) validate() error {
	if b.assetId == 0 {
		return errors.New("asset id is required")
	}
	if b.quantity == 0 {
		return errors.New("quantity must be greater than 0")
	}
	return nil
}

func (b *ModelBuilder) Build() (Model, error) {
	if err := b.validate(); err != nil {
		return Model{}, err
	}
	return Model{
		assetId:  b.assetId,
		quantity: b.quantity,
		ownerId:  b.ownerId,
		flag:     b.flag,
	}, nil
}

// MustBuild builds the model, panicking on validation error.
// Use only for trusted internal data (e.g., from database entities).
func (b *ModelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}

// Clone creates a copy of the Model with modifications
func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
		assetId:  m.assetId,
		quantity: m.quantity,
		ownerId:  m.ownerId,
		flag:     m.flag,
	}
}
