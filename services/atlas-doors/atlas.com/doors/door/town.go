package door

import _map "github.com/Chronicle20/atlas/libs/atlas-constants/map"

// noMap is the MapleStory WZ "no override" sentinel (999999999).
// atlas-data reader emits this as the default for forcedReturnMapId when absent
// (reader.go line 96: GetIntegerWithDefault("forcedReturn", 999999999)).
// It equals _map.EmptyMapId, confirmed in libs/atlas-constants/map/constants.go.
const noMap = _map.EmptyMapId

// HasValidReturn reports whether there is at least one valid return destination.
// Both returnMapId and forcedReturnMapId are EmptyMapId (999999999) when there is
// no return. Note: returnMapId defaults to 0 in atlas-data when absent, and map 0
// (MapleRoadEntranceMushroomTownTrainingCamp1Id) is a real map, so 0 is considered
// valid here — callers must not pass 0 unless it genuinely represents that map.
func HasValidReturn(returnMapId, forcedReturnMapId _map.Id) bool {
	return ResolveTownMap(returnMapId, forcedReturnMapId) != noMap
}

// ResolveTownMap picks the effective town map for a mystic door's return portal.
// forcedReturnMapId wins when it is a real map (not the EmptyMapId sentinel and not 0);
// otherwise returnMapId is used as-is.
func ResolveTownMap(returnMapId, forcedReturnMapId _map.Id) _map.Id {
	if forcedReturnMapId != noMap && forcedReturnMapId != 0 {
		return forcedReturnMapId
	}
	return returnMapId
}
