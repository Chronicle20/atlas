package equipable

import (
	"atlas-consumables/asset"
	"atlas-consumables/kafka/message/compartment"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type Change func(b *asset.ModelBuilder)

func modifyEquipmentProvider(characterId uint32, transactionId uuid.UUID, a asset.Model) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(a.Id()))
	value := &compartment.Command[compartment.ModifyEquipmentCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: byte(inventory.TypeValueEquip),
		Type:          compartment.CommandModifyEquipment,
		Body: compartment.ModifyEquipmentCommandBody{
			AssetId:        a.Id(),
			Strength:       a.Strength(),
			Dexterity:      a.Dexterity(),
			Intelligence:   a.Intelligence(),
			Luck:           a.Luck(),
			Hp:             a.Hp(),
			Mp:             a.Mp(),
			WeaponAttack:   a.WeaponAttack(),
			MagicAttack:    a.MagicAttack(),
			WeaponDefense:  a.WeaponDefense(),
			MagicDefense:   a.MagicDefense(),
			Accuracy:       a.Accuracy(),
			Avoidability:   a.Avoidability(),
			Hands:          a.Hands(),
			Speed:          a.Speed(),
			Jump:           a.Jump(),
			Slots:          a.Slots(),
			Locked:         a.Locked(),
			Spikes:         a.Spikes(),
			KarmaUsed:      a.KarmaUsed(),
			Cold:           a.Cold(),
			CanBeTraded:    a.CanBeTraded(),
			LevelType:      a.LevelType(),
			Level:          a.Level(),
			Experience:     a.Experience(),
			HammersApplied: a.HammersApplied(),
			Expiration:     a.Expiration(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
