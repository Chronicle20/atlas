package movement

import model2 "atlas-channel/socket/model"

func appendElement(mv *model2.Movement, elem Element) {
	mv.Elements = append(mv.Elements, &model2.Element{
		StartX:      elem.StartX,
		StartY:      elem.StartY,
		BMoveAction: elem.MoveAction,
		BStat:       elem.Stat,
		X:           elem.X,
		Y:           elem.Y,
		Vx:          elem.VX,
		Vy:          elem.VY,
		Fh:          elem.FH,
		FhFallStart: elem.FHFallStart,
		XOffset:     elem.XOffset,
		YOffset:     elem.YOffset,
		TElapse:     elem.TimeElapsed,
		ElemType:    elem.TypeVal,
	})
}

func ProduceMovementForSocket(movement Movement) *model2.Movement {
	mv := model2.Movement{StartX: movement.StartX, StartY: movement.StartY}
	for _, elem := range movement.Elements {
		switch elem.TypeStr {
		case MovementTypeNormal:
			mv.Elements = append(mv.Elements, &model2.NormalElement{
				Element: model2.Element{
					StartX:      elem.StartX,
					StartY:      elem.StartY,
					BMoveAction: elem.MoveAction,
					BStat:       elem.Stat,
					X:           elem.X,
					Y:           elem.Y,
					Vx:          elem.VX,
					Vy:          elem.VY,
					Fh:          elem.FH,
					FhFallStart: elem.FHFallStart,
					XOffset:     elem.XOffset,
					YOffset:     elem.YOffset,
					TElapse:     elem.TimeElapsed,
					ElemType:    elem.TypeVal,
				},
			})
		case MovementTypeTeleport:
			mv.Elements = append(mv.Elements, &model2.TeleportElement{
				Element: model2.Element{
					StartX:      elem.StartX,
					StartY:      elem.StartY,
					BMoveAction: elem.MoveAction,
					BStat:       elem.Stat,
					X:           elem.X,
					Y:           elem.Y,
					Vx:          elem.VX,
					Vy:          elem.VY,
					Fh:          elem.FH,
					FhFallStart: elem.FHFallStart,
					XOffset:     elem.XOffset,
					YOffset:     elem.YOffset,
					TElapse:     elem.TimeElapsed,
					ElemType:    elem.TypeVal,
				},
			})
		case MovementTypeStartFallDown:
			mv.Elements = append(mv.Elements, &model2.StartFallDownElement{
				Element: model2.Element{
					StartX:      elem.StartX,
					StartY:      elem.StartY,
					BMoveAction: elem.MoveAction,
					BStat:       elem.Stat,
					X:           elem.X,
					Y:           elem.Y,
					Vx:          elem.VX,
					Vy:          elem.VY,
					Fh:          elem.FH,
					FhFallStart: elem.FHFallStart,
					XOffset:     elem.XOffset,
					YOffset:     elem.YOffset,
					TElapse:     elem.TimeElapsed,
					ElemType:    elem.TypeVal,
				},
			})
		case MovementTypeFlyingBlock:
			mv.Elements = append(mv.Elements, &model2.FlyingBlockElement{
				Element: model2.Element{
					StartX:      elem.StartX,
					StartY:      elem.StartY,
					BMoveAction: elem.MoveAction,
					BStat:       elem.Stat,
					X:           elem.X,
					Y:           elem.Y,
					Vx:          elem.VX,
					Vy:          elem.VY,
					Fh:          elem.FH,
					FhFallStart: elem.FHFallStart,
					XOffset:     elem.XOffset,
					YOffset:     elem.YOffset,
					TElapse:     elem.TimeElapsed,
					ElemType:    elem.TypeVal,
				},
			})
		case MovementTypeJump:
			mv.Elements = append(mv.Elements, &model2.JumpElement{
				Element: model2.Element{
					StartX:      elem.StartX,
					StartY:      elem.StartY,
					BMoveAction: elem.MoveAction,
					BStat:       elem.Stat,
					X:           elem.X,
					Y:           elem.Y,
					Vx:          elem.VX,
					Vy:          elem.VY,
					Fh:          elem.FH,
					FhFallStart: elem.FHFallStart,
					XOffset:     elem.XOffset,
					YOffset:     elem.YOffset,
					TElapse:     elem.TimeElapsed,
					ElemType:    elem.TypeVal,
				},
			})
		case MovementTypeStatChange:
			mv.Elements = append(mv.Elements, &model2.StatChangeElement{
				Element: model2.Element{
					StartX:      elem.StartX,
					StartY:      elem.StartY,
					BMoveAction: elem.MoveAction,
					BStat:       elem.Stat,
					X:           elem.X,
					Y:           elem.Y,
					Vx:          elem.VX,
					Vy:          elem.VY,
					Fh:          elem.FH,
					FhFallStart: elem.FHFallStart,
					XOffset:     elem.XOffset,
					YOffset:     elem.YOffset,
					TElapse:     elem.TimeElapsed,
					ElemType:    elem.TypeVal,
				},
			})
		default:
			appendElement(&mv, elem)
		}
	}
	return &mv
}
