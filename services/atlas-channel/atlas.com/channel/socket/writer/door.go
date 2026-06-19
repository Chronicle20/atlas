package writer

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	doorcb "github.com/Chronicle20/atlas/libs/atlas-packet/door/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// SpawnDoorBody builds the SpawnDoor (SPAWN_DOOR) encode body. ownerId is the
// door owner character id (spawnDoor(ownerid)). x/y is the door position
// on the field it is announced to (area position for the area field). launched
// marks a late-join re-spawn (true) vs a first deploy (false).
func SpawnDoorBody(ownerId uint32, x, y int16, launched bool) packet.Encode {
	return doorcb.NewSpawnDoor(ownerId, x, y, launched).Encode
}

// RemoveDoorBody builds the RemoveDoor (REMOVE_DOOR) encode body for the
// area-side door removal. ownerId is the door owner character id.
func RemoveDoorBody(ownerId uint32) packet.Encode {
	return doorcb.NewRemoveDoor(ownerId).Encode
}

// SpawnPortalBody builds the SpawnPortal (SPAWN_PORTAL) encode body — the
// minimap door indicator. fromMapId/toMapId follow the reference client's spawnPortal(townId,
// targetId) wire order (the announcing side's from-map then to-map). x/y is the
// indicator position on the announcing field.
func SpawnPortalBody(fromMapId, toMapId _map.Id, x, y int16) packet.Encode {
	return doorcb.NewSpawnPortal(fromMapId, toMapId, x, y).Encode
}

// RemoveTownDoorBody builds the RemoveTownDoor encode body — the 8-byte
// SPAWN_PORTAL clear (writeInt(NONE) x2, NO position) used for town-side door
// removal. Do NOT substitute SpawnPortal(NONE,NONE,0,0): that emits 12 bytes
// and corrupts the client read cursor.
func RemoveTownDoorBody() packet.Encode {
	return doorcb.NewRemoveTownDoor().Encode
}
