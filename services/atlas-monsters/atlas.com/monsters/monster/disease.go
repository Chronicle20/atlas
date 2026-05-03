package monster

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// debuffWireValue returns the nValue to ship in an APPLY command for a mob
// debuff skill. v83 wire convention (per Cosmic's giveDebuff): magnitude-
// bearing diseases carry their value in the WZ `x` attribute and pass through
// unchanged, while stat-flag diseases (SEAL/DARKNESS/CURSE/etc.) have no `x`
// in the WZ and need a literal 1 — the client treats nValue==0 as "stat not
// actually applied" and suppresses the icon plus flag-gated effects.
func debuffWireValue(skillId uint16, x int32) int32 {
	switch skillId {
	case monster2.SkillTypePoison,
		monster2.SkillTypeSlow,
		monster2.SkillTypeStopPotion,
		monster2.SkillTypeStopMotion:
		return x
	default:
		if x == 0 {
			return 1
		}
		return x
	}
}

const (
	EnvCommandTopicCharacterBuff = "COMMAND_TOPIC_CHARACTER_BUFF"
	EnvCommandTopicPortal        = "COMMAND_TOPIC_PORTAL"
)

type buffCommand[E any] struct {
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	MapId       _map.Id    `json:"mapId"`
	Instance    uuid.UUID  `json:"instance"`
	CharacterId uint32     `json:"characterId"`
	Type        string     `json:"type"`
	Body        E          `json:"body"`
}

type applyDiseaseBody struct {
	FromId   uint32       `json:"fromId"`
	SourceId int32        `json:"sourceId"`
	Level    byte         `json:"level"`
	Duration int32        `json:"duration"`
	Changes  []statChange `json:"changes"`
}

type cancelAllBuffsBody struct {
}

type statChange struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

func applyDiseaseCommandProvider(f field.Model, characterId uint32, skillId uint16, skillLevel uint16, diseaseName string, value int32, duration int32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value2 := &buffCommand[applyDiseaseBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        "APPLY",
		Body: applyDiseaseBody{
			FromId:   0,
			SourceId: int32(skillId),
			Level:    byte(skillLevel),
			Duration: duration,
			Changes:  []statChange{{Type: diseaseName, Amount: value}},
		},
	}
	return producer.SingleMessageProvider(key, value2)
}

type warpCommand struct {
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	Type      string     `json:"type"`
	Body      warpBody   `json:"body"`
}

type warpBody struct {
	CharacterId uint32  `json:"characterId"`
	TargetMapId _map.Id `json:"targetMapId"`
}

func warpCommandProvider(f field.Model, characterId uint32, targetMapId _map.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &warpCommand{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		Type:      "WARP",
		Body: warpBody{
			CharacterId: characterId,
			TargetMapId: targetMapId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func cancelAllBuffsCommandProvider(f field.Model, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &buffCommand[cancelAllBuffsBody]{
		WorldId:     f.WorldId(),
		ChannelId:   f.ChannelId(),
		MapId:       f.MapId(),
		Instance:    f.Instance(),
		CharacterId: characterId,
		Type:        "CANCEL_ALL",
		Body:        cancelAllBuffsBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
