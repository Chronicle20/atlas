package purchaserecord

import "errors"

// ErrInvalidSerialNumber is returned when the serialNumber is invalid (zero)
var ErrInvalidSerialNumber = errors.New("serialNumber must not be zero")

// modelBuilder is a builder for the Model
type modelBuilder struct {
	serialNumber uint32
	purchased    bool
	count        uint32
}

// NewModelBuilder creates a new modelBuilder with required fields
func NewModelBuilder(serialNumber uint32) *modelBuilder {
	return &modelBuilder{
		serialNumber: serialNumber,
	}
}

// CloneModel creates a builder from this model
func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
		serialNumber: m.serialNumber,
		purchased:    m.purchased,
		count:        m.count,
	}
}

// SetPurchased sets whether this serial number has been purchased
func (b *modelBuilder) SetPurchased(purchased bool) *modelBuilder {
	b.purchased = purchased
	return b
}

// SetCount sets the purchased count for this builder
func (b *modelBuilder) SetCount(count uint32) *modelBuilder {
	b.count = count
	return b
}

// Build creates a Model from this builder
func (b *modelBuilder) Build() (Model, error) {
	if b.serialNumber == 0 {
		return Model{}, ErrInvalidSerialNumber
	}
	return Model{
		serialNumber: b.serialNumber,
		purchased:    b.purchased,
		count:        b.count,
	}, nil
}

// MustBuild creates a Model from this builder and panics if validation fails
func (b *modelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
