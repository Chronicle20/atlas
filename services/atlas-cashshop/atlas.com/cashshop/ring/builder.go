package ring

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Builder is a builder for creating Model instances.
type Builder struct {
	id                 uuid.UUID
	pairId             uuid.UUID
	characterId        uint32
	partnerCharacterId uint32
	assetId            uint32
	itemTemplateId     uint32
	ringType           Type
	state              State
	createdAt          time.Time
}

// NewBuilder creates a new Builder.
func NewBuilder() *Builder {
	return &Builder{
		id:        uuid.New(),
		state:     StateActive,
		createdAt: time.Now(),
	}
}

// SetId sets the ring half's own id.
func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

// SetPairId sets the id shared by both halves of the pair.
func (b *Builder) SetPairId(pairId uuid.UUID) *Builder {
	b.pairId = pairId
	return b
}

// SetCharacterId sets the character this half belongs to.
func (b *Builder) SetCharacterId(characterId uint32) *Builder {
	b.characterId = characterId
	return b
}

// SetPartnerCharacterId sets the other half's character.
func (b *Builder) SetPartnerCharacterId(partnerCharacterId uint32) *Builder {
	b.partnerCharacterId = partnerCharacterId
	return b
}

// SetAssetId sets the cash asset backing this half.
func (b *Builder) SetAssetId(assetId uint32) *Builder {
	b.assetId = assetId
	return b
}

// SetItemTemplateId sets the item template this half was purchased as.
func (b *Builder) SetItemTemplateId(itemTemplateId uint32) *Builder {
	b.itemTemplateId = itemTemplateId
	return b
}

// SetType sets the pair type (couple/friendship).
func (b *Builder) SetType(ringType Type) *Builder {
	b.ringType = ringType
	return b
}

// SetState sets the half's current state.
func (b *Builder) SetState(state State) *Builder {
	b.state = state
	return b
}

// SetCreatedAt sets when this half was created.
func (b *Builder) SetCreatedAt(createdAt time.Time) *Builder {
	b.createdAt = createdAt
	return b
}

// Build validates and constructs the Model.
func (b *Builder) Build() (Model, error) {
	if err := b.validate(); err != nil {
		return Model{}, err
	}
	return Model{
		id:                 b.id,
		pairId:             b.pairId,
		characterId:        b.characterId,
		partnerCharacterId: b.partnerCharacterId,
		assetId:            b.assetId,
		itemTemplateId:     b.itemTemplateId,
		ringType:           b.ringType,
		state:              b.state,
		createdAt:          b.createdAt,
	}, nil
}

func (b *Builder) validate() error {
	if b.characterId == 0 {
		return errors.New("characterId is required")
	}
	if b.partnerCharacterId == 0 {
		return errors.New("partnerCharacterId is required")
	}
	if b.ringType != TypeCouple && b.ringType != TypeFriendship {
		return errors.New("ringType must be COUPLE or FRIENDSHIP")
	}
	return nil
}
