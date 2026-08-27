package ring

import (
	"errors"

	"github.com/google/uuid"
)

// ErrInvalidId is returned when the id is invalid (nil UUID)
var ErrInvalidId = errors.New("id must not be zero UUID")

// ErrInvalidPairId is returned when the pairId is invalid (nil UUID)
var ErrInvalidPairId = errors.New("pairId must not be zero UUID")

// modelBuilder is a builder for the Model
type modelBuilder struct {
	id                 uuid.UUID
	pairId             uuid.UUID
	characterId        uint32
	partnerCharacterId uint32
	itemTemplateId     uint32
	ringType           Type
	state              State
	cashId             int64
	partnerCashId      int64
	partnerName        string
}

// NewModelBuilder creates a new modelBuilder with required fields
func NewModelBuilder(id uuid.UUID, pairId uuid.UUID) *modelBuilder {
	return &modelBuilder{
		id:     id,
		pairId: pairId,
	}
}

// CloneModel creates a builder from this model
func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
		id:                 m.id,
		pairId:             m.pairId,
		characterId:        m.characterId,
		partnerCharacterId: m.partnerCharacterId,
		itemTemplateId:     m.itemTemplateId,
		ringType:           m.ringType,
		state:              m.state,
		cashId:             m.cashId,
		partnerCashId:      m.partnerCashId,
		partnerName:        m.partnerName,
	}
}

// SetCharacterId sets the owning character id for this builder
func (b *modelBuilder) SetCharacterId(characterId uint32) *modelBuilder {
	b.characterId = characterId
	return b
}

// SetPartnerCharacterId sets the partner character id for this builder
func (b *modelBuilder) SetPartnerCharacterId(partnerCharacterId uint32) *modelBuilder {
	b.partnerCharacterId = partnerCharacterId
	return b
}

// SetItemTemplateId sets the item template id for this builder
func (b *modelBuilder) SetItemTemplateId(itemTemplateId uint32) *modelBuilder {
	b.itemTemplateId = itemTemplateId
	return b
}

// SetType sets the ring pair type for this builder
func (b *modelBuilder) SetType(t Type) *modelBuilder {
	b.ringType = t
	return b
}

// SetState sets the pair-half state for this builder
func (b *modelBuilder) SetState(s State) *modelBuilder {
	b.state = s
	return b
}

// SetCashId sets this half's own locker asset cash id for this builder
func (b *modelBuilder) SetCashId(cashId int64) *modelBuilder {
	b.cashId = cashId
	return b
}

// SetPartnerCashId sets the sibling half's cash id for this builder
func (b *modelBuilder) SetPartnerCashId(partnerCashId int64) *modelBuilder {
	b.partnerCashId = partnerCashId
	return b
}

// SetPartnerName sets the resolved partner character name for this builder
func (b *modelBuilder) SetPartnerName(partnerName string) *modelBuilder {
	b.partnerName = partnerName
	return b
}

// Build creates a Model from this builder
func (b *modelBuilder) Build() (Model, error) {
	if b.id == uuid.Nil {
		return Model{}, ErrInvalidId
	}
	if b.pairId == uuid.Nil {
		return Model{}, ErrInvalidPairId
	}
	return Model{
		id:                 b.id,
		pairId:             b.pairId,
		characterId:        b.characterId,
		partnerCharacterId: b.partnerCharacterId,
		itemTemplateId:     b.itemTemplateId,
		ringType:           b.ringType,
		state:              b.state,
		cashId:             b.cashId,
		partnerCashId:      b.partnerCashId,
		partnerName:        b.partnerName,
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
