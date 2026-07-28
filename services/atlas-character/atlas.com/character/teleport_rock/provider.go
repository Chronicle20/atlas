package teleport_rock

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

func modelFromEntities(characterId uint32, es []entity) Model {
	var regular, vip []_map.Id
	for _, e := range es {
		if e.ListType == ListTypeVip {
			vip = append(vip, e.MapId)
		} else {
			regular = append(regular, e.MapId)
		}
	}
	return NewBuilder().
		SetCharacterId(characterId).
		SetRegular(regular).
		SetVip(vip).
		Build()
}
