package monster

import (
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

func Move(ma _map.Model, uniqueId uint32, observerId uint32, skillPossible bool, skill int8, skillId int16, skillLevel int16, multiTarget model2.MultiTargetForBall, randTimes model2.RandTimeForAreaAttack, mm model2.Movement) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(uniqueId))
	ps := make([]position, 0)
	for _, p := range multiTarget.Targets {
		ps = append(ps, position{p.X(), p.Y()})
	}

	m := movement{StartX: mm.StartX, StartY: mm.StartY}
	for _, elem := range mm.Elements {
		elemType := ""
		if val, ok := elem.(*model2.NormalElement); ok {
			elemType = MovementTypeNormal
			m.Elements = append(m.Elements, element{
				TypeStr:     elemType,
				TypeVal:     val.ElemType,
				StartX:      val.StartX,
				StartY:      val.StartY,
				MoveAction:  val.BMoveAction,
				Stat:        val.BStat,
				X:           val.X,
				Y:           val.Y,
				VX:          val.Vx,
				VY:          val.Vy,
				FH:          val.Fh,
				FHFallStart: val.FhFallStart,
				XOffset:     val.XOffset,
				YOffset:     val.YOffset,
				TimeElapsed: val.TElapse,
			})
		}
		if val, ok := elem.(*model2.TeleportElement); ok {
			elemType = MovementTypeTeleport
			m.Elements = append(m.Elements, element{
				TypeStr:     elemType,
				TypeVal:     val.ElemType,
				StartX:      val.StartX,
				StartY:      val.StartY,
				MoveAction:  val.BMoveAction,
				Stat:        val.BStat,
				X:           val.X,
				Y:           val.Y,
				VX:          val.Vx,
				VY:          val.Vy,
				FH:          val.Fh,
				FHFallStart: val.FhFallStart,
				XOffset:     val.XOffset,
				YOffset:     val.YOffset,
				TimeElapsed: val.TElapse,
			})
		}
		if val, ok := elem.(*model2.StartFallDownElement); ok {
			elemType = MovementTypeStartFallDown
			m.Elements = append(m.Elements, element{
				TypeStr:     elemType,
				TypeVal:     val.ElemType,
				StartX:      val.StartX,
				StartY:      val.StartY,
				MoveAction:  val.BMoveAction,
				Stat:        val.BStat,
				X:           val.X,
				Y:           val.Y,
				VX:          val.Vx,
				VY:          val.Vy,
				FH:          val.Fh,
				FHFallStart: val.FhFallStart,
				XOffset:     val.XOffset,
				YOffset:     val.YOffset,
				TimeElapsed: val.TElapse,
			})
		}
		if val, ok := elem.(*model2.FlyingBlockElement); ok {
			elemType = MovementTypeFlyingBlock
			m.Elements = append(m.Elements, element{
				TypeStr:     elemType,
				TypeVal:     val.ElemType,
				StartX:      val.StartX,
				StartY:      val.StartY,
				MoveAction:  val.BMoveAction,
				Stat:        val.BStat,
				X:           val.X,
				Y:           val.Y,
				VX:          val.Vx,
				VY:          val.Vy,
				FH:          val.Fh,
				FHFallStart: val.FhFallStart,
				XOffset:     val.XOffset,
				YOffset:     val.YOffset,
				TimeElapsed: val.TElapse,
			})
		}
		if val, ok := elem.(*model2.JumpElement); ok {
			elemType = MovementTypeJump
			m.Elements = append(m.Elements, element{
				TypeStr:     elemType,
				TypeVal:     val.ElemType,
				StartX:      val.StartX,
				StartY:      val.StartY,
				MoveAction:  val.BMoveAction,
				Stat:        val.BStat,
				X:           val.X,
				Y:           val.Y,
				VX:          val.Vx,
				VY:          val.Vy,
				FH:          val.Fh,
				FHFallStart: val.FhFallStart,
				XOffset:     val.XOffset,
				YOffset:     val.YOffset,
				TimeElapsed: val.TElapse,
			})
		}
		if val, ok := elem.(*model2.StatChangeElement); ok {
			elemType = MovementTypeStatChange
			m.Elements = append(m.Elements, element{
				TypeStr:     elemType,
				TypeVal:     val.ElemType,
				StartX:      val.StartX,
				StartY:      val.StartY,
				MoveAction:  val.BMoveAction,
				Stat:        val.BStat,
				X:           val.X,
				Y:           val.Y,
				VX:          val.Vx,
				VY:          val.Vy,
				FH:          val.Fh,
				FHFallStart: val.FhFallStart,
				XOffset:     val.XOffset,
				YOffset:     val.YOffset,
				TimeElapsed: val.TElapse,
			})
		}
		if val, ok := elem.(*model2.Element); ok {
			m.Elements = append(m.Elements, element{
				TypeStr:     elemType,
				TypeVal:     val.ElemType,
				StartX:      val.StartX,
				StartY:      val.StartY,
				MoveAction:  val.BMoveAction,
				Stat:        val.BStat,
				X:           val.X,
				Y:           val.Y,
				VX:          val.Vx,
				VY:          val.Vy,
				FH:          val.Fh,
				FHFallStart: val.FhFallStart,
				XOffset:     val.XOffset,
				YOffset:     val.YOffset,
				TimeElapsed: val.TElapse,
			})
		}

	}

	value := &movementCommand{
		WorldId:       byte(ma.WorldId()),
		ChannelId:     byte(ma.ChannelId()),
		UniqueId:      uniqueId,
		ObserverId:    observerId,
		SkillPossible: skillPossible,
		Skill:         skill,
		SkillId:       skillId,
		SkillLevel:    skillLevel,
		MultiTarget:   ps,
		RandomTimes:   randTimes.Times,
		Movement:      m,
	}
	return producer.SingleMessageProvider(key, value)
}
