package item

import (
	"errors"
	"time"
)

// ErrInvalidId is returned when the id is invalid (zero)
var ErrInvalidId = errors.New("id must be greater than 0")

// modelBuilder is used to build Model instances
type builder struct {
	id          uint32
	cashId      int64
	templateId  uint32
	quantity    uint32
	flag        uint16
	purchasedBy uint32
	expiration  time.Time
}

// NewBuilder creates a new builder
func NewBuilder() *builder {
	return &builder{}
}

// CloneModel creates a new modelBuilder with values from the given Model
func CloneModel(m Model) *builder {
	return &builder{
		id:          m.id,
		cashId:      m.cashId,
		templateId:  m.templateId,
		quantity:    m.quantity,
		flag:        m.flag,
		purchasedBy: m.purchasedBy,
		expiration:  m.expiration,
	}
}

// SetId sets the id for the modelBuilder
func (b *builder) SetId(id uint32) *builder {
	b.id = id
	return b
}

// SetCashId sets the cashId for the modelBuilder
func (b *builder) SetCashId(cashId int64) *builder {
	b.cashId = cashId
	return b
}

// SetTemplateId sets the templateId for the modelBuilder
func (b *builder) SetTemplateId(templateId uint32) *builder {
	b.templateId = templateId
	return b
}

// SetQuantity sets the quantity for the modelBuilder
func (b *builder) SetQuantity(quantity uint32) *builder {
	b.quantity = quantity
	return b
}

// SetFlag sets the flag for the modelBuilder
func (b *builder) SetFlag(flag uint16) *builder {
	b.flag = flag
	return b
}

// SetPurchasedBy sets the purchasedBy for the modelBuilder
func (b *builder) SetPurchasedBy(purchasedBy uint32) *builder {
	b.purchasedBy = purchasedBy
	return b
}

// SetExpiration sets the expiration for the modelBuilder
func (b *builder) SetExpiration(expiration time.Time) *builder {
	b.expiration = expiration
	return b
}

// Build creates a new Model instance with the builder's values
func (b *builder) Build() (Model, error) {
	if b.id == 0 {
		return Model{}, ErrInvalidId
	}
	return Model{
		id:          b.id,
		cashId:      b.cashId,
		templateId:  b.templateId,
		quantity:    b.quantity,
		flag:        b.flag,
		purchasedBy: b.purchasedBy,
		expiration:  b.expiration,
	}, nil
}

// MustBuild creates a new Model instance and panics if validation fails
func (b *builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
