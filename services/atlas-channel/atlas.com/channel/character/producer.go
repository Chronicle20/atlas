package character

import (
	model2 "atlas-channel/socket/model"
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func move(tenant tenant.Model, worldId byte, channelId byte, mapId uint32, characterId uint32, mm model2.Movement) model.Provider[[]kafka.Message] {
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
		Tenant:      tenant,
		WorldId:     worldId,
		ChannelId:   channelId,
		MapId:       mapId,
		CharacterId: characterId,
		Movement:    m,
	}
	return producer.SingleMessageProvider(key, value)
}
