package frederick

import (
	"atlas-merchant/kafka/message/asset"

	"github.com/google/uuid"
)

type ItemModel struct {
	id           uuid.UUID
	characterId  uint32
	itemId       uint32
	itemType     byte
	quantity     uint16
	itemSnapshot asset.AssetData
}

func (m ItemModel) Id() uuid.UUID         { return m.id }
func (m ItemModel) CharacterId() uint32    { return m.characterId }
func (m ItemModel) ItemId() uint32         { return m.itemId }
func (m ItemModel) ItemType() byte         { return m.itemType }
func (m ItemModel) Quantity() uint16       { return m.quantity }
func (m ItemModel) ItemSnapshot() asset.AssetData { return m.itemSnapshot }

func MakeItem(e ItemEntity) (ItemModel, error) {
	return ItemModel{
		id:           e.Id,
		characterId:  e.CharacterId,
		itemId:       e.ItemId,
		itemType:     e.ItemType,
		quantity:     e.Quantity,
		itemSnapshot: e.ItemSnapshot,
	}, nil
}

type MesoModel struct {
	id          uuid.UUID
	characterId uint32
	amount      uint32
}

func (m MesoModel) Id() uuid.UUID      { return m.id }
func (m MesoModel) CharacterId() uint32 { return m.characterId }
func (m MesoModel) Amount() uint32      { return m.amount }

func MakeMeso(e MesoEntity) (MesoModel, error) {
	return MesoModel{
		id:          e.Id,
		characterId: e.CharacterId,
		amount:      e.Amount,
	}, nil
}
