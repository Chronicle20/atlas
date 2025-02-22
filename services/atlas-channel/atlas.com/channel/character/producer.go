package character

import (
	model2 "atlas-channel/socket/model"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func requestDistributeApCommandProvider(m _map.Model, characterId uint32, distributions []DistributePair) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestDistributeApCommandBody]{
		CharacterId: characterId,
		WorldId:     byte(m.WorldId()),
		Type:        CommandRequestDistributeAp,
		Body: requestDistributeApCommandBody{
			Distributions: distributions,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestDistributeSpCommandProvider(m _map.Model, characterId uint32, skillId uint32, amount int8) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestDistributeSpCommandBody]{
		CharacterId: characterId,
		WorldId:     byte(m.WorldId()),
		Type:        CommandRequestDistributeSp,
		Body: requestDistributeSpCommandBody{
			SkillId: skillId,
			Amount:  amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func requestDropMesoCommandProvider(m _map.Model, characterId uint32, amount uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[requestDropMesoCommandBody]{
		CharacterId: characterId,
		WorldId:     byte(m.WorldId()),
		Type:        CommandRequestDropMeso,
		Body: requestDropMesoCommandBody{
			ChannelId: byte(m.ChannelId()),
			MapId:     uint32(m.MapId()),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeHPCommandProvider(m _map.Model, characterId uint32, amount int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[changeHPCommandBody]{
		CharacterId: characterId,
		WorldId:     byte(m.WorldId()),
		Type:        CommandChangeHP,
		Body: changeHPCommandBody{
			ChannelId: byte(m.ChannelId()),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeMPCommandProvider(m _map.Model, characterId uint32, amount int16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[changeMPCommandBody]{
		CharacterId: characterId,
		WorldId:     byte(m.WorldId()),
		Type:        CommandChangeMP,
		Body: changeMPCommandBody{
			ChannelId: byte(m.ChannelId()),
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func move(ma _map.Model, characterId uint32, mm model2.Movement) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))

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
		WorldId:     byte(ma.WorldId()),
		ChannelId:   byte(ma.ChannelId()),
		MapId:       uint32(ma.MapId()),
		CharacterId: characterId,
		Movement:    m,
	}
	return producer.SingleMessageProvider(key, value)
}
