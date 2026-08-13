package buff

import (
	"atlas-channel/data/skill/effect/statup"
	"atlas-channel/kafka/message/buff"

	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func ApplyCommandProvider(f field.Model, characterId uint32, fromId uint32, sourceId int32, level byte, duration int32, statups []statup.Model) model.Provider[[]kafka.Message] {
	changes := make([]buff.StatChange, 0)
	for _, su := range statups {
		changes = append(changes, buff.StatChange{
			Type:   su.Mask(),
			Amount: su.Amount(),
		})
	}

	key := producer.CreateKey(int(characterId))
	value := &buff.Command[buff.ApplyCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        buff.CommandTypeApply,
		Body: buff.ApplyCommandBody{
			FromId:   fromId,
			SourceId: sourceId,
			Level:    level,
			Duration: duration,
			Changes:  changes,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// ApplyNoExpiryCommandProvider emits an APPLY carrying the explicit noExpiry
// flag (Duration 0 — atlas-buffs rejects the flag with a nonzero duration).
func ApplyNoExpiryCommandProvider(f field.Model, characterId uint32, fromId uint32, sourceId int32, level byte, statups []statup.Model) model.Provider[[]kafka.Message] {
	changes := make([]buff.StatChange, 0)
	for _, su := range statups {
		changes = append(changes, buff.StatChange{
			Type:   su.Mask(),
			Amount: su.Amount(),
		})
	}

	key := producer.CreateKey(int(characterId))
	value := &buff.Command[buff.ApplyCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        buff.CommandTypeApply,
		Body: buff.ApplyCommandBody{
			FromId:   fromId,
			SourceId: sourceId,
			Level:    level,
			Duration: 0,
			Changes:  changes,
			NoExpiry: true,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func CancelCommandProvider(f field.Model, characterId uint32, sourceId int32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &buff.Command[buff.CancelCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        buff.CommandTypeCancel,
		Body: buff.CancelCommandBody{
			SourceId: sourceId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func CancelByTypesCommandProvider(f field.Model, characterId uint32, types []string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &buff.Command[buff.CancelByTypesCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        buff.CommandTypeCancelByTypes,
		Body: buff.CancelByTypesCommandBody{
			Types: append([]string(nil), types...),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

// ExpireCommandProvider asks atlas-buffs to sweep ONE character's buffs. The
// world rides in the envelope: the channel knows the live session's world, and
// that is authoritative for an in-session character (the fleet sweep instead
// reads world from the registry model). (task-190 FR-2.6)
func ExpireCommandProvider(f field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &buff.Command[buff.ExpireCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        buff.CommandTypeExpire,
		Body:        buff.ExpireCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

// StatValueUpdate is one UPDATE_STAT_VALUE request. Mirrors
// atlas-buffs/character.StatValueUpdate; collected into a struct rather than
// passed positionally because the accumulator-upsert fields (task-216) push
// the argument list past readability.
type StatValueUpdate struct {
	SourceId        int32
	StatType        string
	Operation       string
	Amount          int32
	Cap             int32
	CreateIfMissing bool
	Level           byte
}

func UpdateStatValueCommandProvider(f field.Model, characterId uint32, u StatValueUpdate) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &buff.Command[buff.UpdateStatValueCommandBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        buff.CommandTypeUpdateStatValue,
		Body: buff.UpdateStatValueCommandBody{
			SourceId:        u.SourceId,
			StatType:        u.StatType,
			Operation:       u.Operation,
			Amount:          u.Amount,
			Cap:             u.Cap,
			CreateIfMissing: u.CreateIfMissing,
			Level:           u.Level,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
