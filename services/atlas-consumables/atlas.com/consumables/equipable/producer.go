package equipable

import (
	"atlas-consumables/asset"
	"atlas-consumables/kafka/message/equipable"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

type Change func(b *asset.EquipableReferenceDataBuilder)

func changeEquipableProvider(a asset.Model[asset.EquipableReferenceData]) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(a.Id()))
	value := &equipable.Command[equipable.ChangeBody]{
		Id:   a.ReferenceId(),
		Type: equipable.CommandChange,
		Body: equipable.ChangeBody{
			Strength:       a.ReferenceData().GetStrength(),
			Dexterity:      a.ReferenceData().GetDexterity(),
			Intelligence:   a.ReferenceData().GetIntelligence(),
			Luck:           a.ReferenceData().GetLuck(),
			HP:             a.ReferenceData().GetHP(),
			MP:             a.ReferenceData().GetMP(),
			WeaponAttack:   a.ReferenceData().GetWeaponAttack(),
			MagicAttack:    a.ReferenceData().GetMagicAttack(),
			WeaponDefense:  a.ReferenceData().GetWeaponDefense(),
			MagicDefense:   a.ReferenceData().GetMagicDefense(),
			Accuracy:       a.ReferenceData().GetAccuracy(),
			Avoidability:   a.ReferenceData().GetAvoidability(),
			Hands:          a.ReferenceData().GetHands(),
			Speed:          a.ReferenceData().GetSpeed(),
			Jump:           a.ReferenceData().GetJump(),
			Slots:          a.ReferenceData().GetSlots(),
			OwnerName:      "", //TODO
			Locked:         a.ReferenceData().IsLocked(),
			Spikes:         a.ReferenceData().HasSpikes(),
			KarmaUsed:      a.ReferenceData().IsKarmaUsed(),
			Cold:           a.ReferenceData().IsCold(),
			CanBeTraded:    a.ReferenceData().CanBeTraded(),
			LevelType:      a.ReferenceData().GetLevelType(),
			Level:          a.ReferenceData().GetLevel(),
			Experience:     a.ReferenceData().GetExperience(),
			HammersApplied: a.ReferenceData().GetHammersApplied(),
			Expiration:     a.Expiration(),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
