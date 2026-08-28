package npc

import (
	npcdata "atlas-channel/data/npc"
	"atlas-channel/playernpc"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
)

// ResolveTemplate resolves an inbound client object id to the template
// (script) id of the NPC it names, and errors when no such object is
// present in f.
//
// Inbound NPC action and movement packets arrive from the object's elected
// controller, and since task-251 bug report §5 a controller is granted for
// Player NPCs too. A Player NPC oid is never in atlas-data's per-map life
// list -- the WZ imitate placeholders are filtered out of it and the
// deployed object is not a life entry at all -- so it resolves through
// atlas-player-npcs instead, dispatching on the reserved oid band
// (design D-5).
func ResolveTemplate(l logrus.FieldLogger, ctx context.Context, f field.Model, objectId uint32) (uint32, error) {
	if objectid.IsPlayerNpcObjectId(objectId) {
		n, err := playernpc.NewProcessor(l, ctx).GetInMapByObjectId(f, objectId)
		if err != nil {
			return 0, err
		}
		return n.ScriptId(), nil
	}
	n, err := npcdata.NewProcessor(l, ctx).GetInMapByObjectId(f.MapId(), objectId)
	if err != nil {
		return 0, err
	}
	return n.Template(), nil
}
