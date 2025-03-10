package movement

import model2 "atlas-channel/socket/model"

func ProduceMovementForKafka(mm model2.Movement) Movement {
	m := Movement{StartX: mm.StartX, StartY: mm.StartY}
	for _, elem := range mm.Elements {
		var elemType string
		switch elem.(type) {
		case *model2.NormalElement:
			elemType = MovementTypeNormal
		case *model2.TeleportElement:
			elemType = MovementTypeTeleport
		case *model2.StartFallDownElement:
			elemType = MovementTypeStartFallDown
		case *model2.FlyingBlockElement:
			elemType = MovementTypeFlyingBlock
		case *model2.JumpElement:
			elemType = MovementTypeJump
		case *model2.StatChangeElement:
			elemType = MovementTypeStatChange
		default:
			continue
		}

		appendElementForKafka(&m, elem, elemType)
	}
	return m
}

func appendElementForKafka(m *Movement, elem interface{}, elemType string) {
	switch val := elem.(type) {
	case *model2.NormalElement:
		appendToMovement(m, elemType, val.ElemType, val.StartX, val.StartY, val.BMoveAction, val.BStat, val.X, val.Y, val.Vx, val.Vy, val.Fh, val.FhFallStart, val.XOffset, val.YOffset, val.TElapse)
	case *model2.TeleportElement:
		appendToMovement(m, elemType, val.ElemType, val.StartX, val.StartY, val.BMoveAction, val.BStat, val.X, val.Y, val.Vx, val.Vy, val.Fh, val.FhFallStart, val.XOffset, val.YOffset, val.TElapse)
	case *model2.StartFallDownElement:
		appendToMovement(m, elemType, val.ElemType, val.StartX, val.StartY, val.BMoveAction, val.BStat, val.X, val.Y, val.Vx, val.Vy, val.Fh, val.FhFallStart, val.XOffset, val.YOffset, val.TElapse)
	case *model2.FlyingBlockElement:
		appendToMovement(m, elemType, val.ElemType, val.StartX, val.StartY, val.BMoveAction, val.BStat, val.X, val.Y, val.Vx, val.Vy, val.Fh, val.FhFallStart, val.XOffset, val.YOffset, val.TElapse)
	case *model2.JumpElement:
		appendToMovement(m, elemType, val.ElemType, val.StartX, val.StartY, val.BMoveAction, val.BStat, val.X, val.Y, val.Vx, val.Vy, val.Fh, val.FhFallStart, val.XOffset, val.YOffset, val.TElapse)
	case *model2.StatChangeElement:
		appendToMovement(m, elemType, val.ElemType, val.StartX, val.StartY, val.BMoveAction, val.BStat, val.X, val.Y, val.Vx, val.Vy, val.Fh, val.FhFallStart, val.XOffset, val.YOffset, val.TElapse)
	case *model2.Element:
		appendToMovement(m, elemType, val.ElemType, val.StartX, val.StartY, val.BMoveAction, val.BStat, val.X, val.Y, val.Vx, val.Vy, val.Fh, val.FhFallStart, val.XOffset, val.YOffset, val.TElapse)
	}
}

func appendToMovement(m *Movement, elemType string, typeVal byte, startX int16, startY int16, moveAction byte, stat byte, x int16, y int16, vx int16, vy int16, fh int16, fhFallStart int16, xOffset int16, yOffset int16, timeElapsed int16) {
	m.Elements = append(m.Elements, Element{
		TypeStr:     elemType,
		TypeVal:     typeVal,
		StartX:      startX,
		StartY:      startY,
		MoveAction:  moveAction,
		Stat:        stat,
		X:           x,
		Y:           y,
		VX:          vx,
		VY:          vy,
		FH:          fh,
		FHFallStart: fhFallStart,
		XOffset:     xOffset,
		YOffset:     yOffset,
		TimeElapsed: timeElapsed,
	})
}
