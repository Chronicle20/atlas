package frederick

import (
	"atlas-merchant/kafka/message/asset"
	"errors"

	"github.com/google/uuid"
)

func NewItemBuilder() *ItemModelBuilder {
	return &ItemModelBuilder{}
}

type ItemModelBuilder struct {
	id           uuid.UUID
	characterId  uint32
	itemId       uint32
	itemType     byte
	quantity     uint16
	itemSnapshot asset.AssetData
}

func (b *ItemModelBuilder) SetId(id uuid.UUID) *ItemModelBuilder {
	b.id = id
	return b
}

func (b *ItemModelBuilder) SetCharacterId(characterId uint32) *ItemModelBuilder {
	b.characterId = characterId
	return b
}

func (b *ItemModelBuilder) SetItemId(itemId uint32) *ItemModelBuilder {
	b.itemId = itemId
	return b
}

func (b *ItemModelBuilder) SetItemType(itemType byte) *ItemModelBuilder {
	b.itemType = itemType
	return b
}

func (b *ItemModelBuilder) SetQuantity(quantity uint16) *ItemModelBuilder {
	b.quantity = quantity
	return b
}

func (b *ItemModelBuilder) SetItemSnapshot(itemSnapshot asset.AssetData) *ItemModelBuilder {
	b.itemSnapshot = itemSnapshot
	return b
}

func (b *ItemModelBuilder) Build() (ItemModel, error) {
	if b.id == uuid.Nil {
		return ItemModel{}, errors.New("id is required")
	}
	if b.characterId == 0 {
		return ItemModel{}, errors.New("characterId is required")
	}
	if b.itemId == 0 {
		return ItemModel{}, errors.New("itemId is required")
	}
	return ItemModel{
		id:           b.id,
		characterId:  b.characterId,
		itemId:       b.itemId,
		itemType:     b.itemType,
		quantity:     b.quantity,
		itemSnapshot: b.itemSnapshot,
	}, nil
}

func NewMesoBuilder() *MesoModelBuilder {
	return &MesoModelBuilder{}
}

type MesoModelBuilder struct {
	id          uuid.UUID
	characterId uint32
	amount      uint32
}

func (b *MesoModelBuilder) SetId(id uuid.UUID) *MesoModelBuilder {
	b.id = id
	return b
}

func (b *MesoModelBuilder) SetCharacterId(characterId uint32) *MesoModelBuilder {
	b.characterId = characterId
	return b
}

func (b *MesoModelBuilder) SetAmount(amount uint32) *MesoModelBuilder {
	b.amount = amount
	return b
}

func (b *MesoModelBuilder) Build() (MesoModel, error) {
	if b.id == uuid.Nil {
		return MesoModel{}, errors.New("id is required")
	}
	if b.characterId == 0 {
		return MesoModel{}, errors.New("characterId is required")
	}
	return MesoModel{
		id:          b.id,
		characterId: b.characterId,
		amount:      b.amount,
	}, nil
}
