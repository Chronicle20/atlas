package item

import (
	"errors"
	"time"
)

// ErrInvalidId is returned when the id is invalid (zero)
var ErrInvalidId = errors.New("id must be greater than 0")

// modelBuilder is used to build Model instances
type modelBuilder struct {
	id          uint32
	cashId      int64
	templateId  uint32
	quantity    uint32
	flag        uint16
	purchasedBy uint32
	expiration  time.Time
}

// NewModelBuilder creates a new modelBuilder
func NewModelBuilder() *modelBuilder {
	return &modelBuilder{}
}

// CloneModel creates a new modelBuilder with values from the given Model
func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
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
func (b *modelBuilder) SetId(id uint32) *modelBuilder {
	b.id = id
	return b
}

// SetCashId sets the cashId for the modelBuilder
func (b *modelBuilder) SetCashId(cashId int64) *modelBuilder {
	b.cashId = cashId
	return b
}

// SetTemplateId sets the templateId for the modelBuilder
func (b *modelBuilder) SetTemplateId(templateId uint32) *modelBuilder {
	b.templateId = templateId
	return b
}

// SetQuantity sets the quantity for the modelBuilder
func (b *modelBuilder) SetQuantity(quantity uint32) *modelBuilder {
	b.quantity = quantity
	return b
}

// SetFlag sets the flag for the modelBuilder
func (b *modelBuilder) SetFlag(flag uint16) *modelBuilder {
	b.flag = flag
	return b
}

// SetPurchasedBy sets the purchasedBy for the modelBuilder
func (b *modelBuilder) SetPurchasedBy(purchasedBy uint32) *modelBuilder {
	b.purchasedBy = purchasedBy
	return b
}

// SetExpiration sets the expiration for the modelBuilder
func (b *modelBuilder) SetExpiration(expiration time.Time) *modelBuilder {
	b.expiration = expiration
	return b
}

// Build creates a new Model instance with the builder's values
func (b *modelBuilder) Build() (Model, error) {
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
func (b *modelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
