package monster

import (
	"atlas-channel/movement"
	model2 "atlas-channel/socket/model"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func damageCommandProvider(m _map.Model, monsterId uint32, characterId uint32, damage uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(monsterId))
	value := &command[damageCommandBody]{
		WorldId:   byte(m.WorldId()),
		ChannelId: byte(m.ChannelId()),
		MonsterId: monsterId,
		Type:      CommandTypeDamage,
		Body: damageCommandBody{
			CharacterId: characterId,
			Damage:      damage,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func Move(uniqueId uint32, observerId uint32, skillPossible bool, skill int8, skillId int16, skillLevel int16, multiTarget model2.MultiTargetForBall, randTimes model2.RandTimeForAreaAttack, mm model2.Movement) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(uniqueId))
	ps := make([]position, 0)
	for _, p := range multiTarget.Targets {
		ps = append(ps, position{p.X(), p.Y()})
	}

	value := &movementCommand{
		UniqueId:      uniqueId,
		ObserverId:    observerId,
		SkillPossible: skillPossible,
		Skill:         skill,
		SkillId:       skillId,
		SkillLevel:    skillLevel,
		MultiTarget:   ps,
		RandomTimes:   randTimes.Times,
		Movement:      movement.ProduceMovementForKafka(mm),
	}
	return producer.SingleMessageProvider(key, value)
}
