package visit

import (
	"time"

	_map "github.com/Chronicle20/atlas-constants/map"
)

type Visit struct {
	characterId    uint32
	mapId          _map.Id
	firstVisitedAt time.Time
}

func (v Visit) CharacterId() uint32 {
	return v.characterId
}

func (v Visit) MapId() _map.Id {
	return v.mapId
}

func (v Visit) FirstVisitedAt() time.Time {
	return v.firstVisitedAt
}
