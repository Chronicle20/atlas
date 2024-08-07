package monster

import (
	model2 "atlas-channel/socket/model"
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func Move(tenant tenant.Model, worldId byte, channelId byte, uniqueId uint32, observerId uint32, skillPossible bool,
	skill int8, skillId int16, skillLevel int16, multiTarget model2.MultiTargetForBall,
	randTimes model2.RandTimeForAreaAttack, rawMovement []byte) model.Provider[[]kafka.Message] {

	key := producer.CreateKey(int(uniqueId))
	ps := make([]position, 0)
	for _, p := range multiTarget.Targets {
		ps = append(ps, position{p.X(), p.Y()})
	}
	value := &movementCommand{
		Tenant:        tenant,
		WorldId:       worldId,
		ChannelId:     channelId,
		UniqueId:      uniqueId,
		ObserverId:    observerId,
		SkillPossible: skillPossible,
		Skill:         skill,
		SkillId:       skillId,
		SkillLevel:    skillLevel,
		MultiTarget:   ps,
		RandomTimes:   randTimes.Times,
		RawMovement:   rawMovement,
	}
	return producer.SingleMessageProvider(key, value)
}
