package writer

import (
	"context"

	"github.com/sirupsen/logrus"

	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// CatchMonsterWithItemBody encodes the clientbound CATCH_MONSTER_WITH_ITEM
// packet, which plays a capture-by-item effect on a targeted mob. The leading
// uniqueId is consumed by CMobPool::OnMobPacket before dispatch, so the mob
// must still exist client-side when this arrives (task-212 design §4.2).
func CatchMonsterWithItemBody(uniqueId uint32, itemId int32, result byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return monsterpkt.NewCatchMonsterWithItem(uniqueId, itemId, result).Encode(l, ctx)(options)
		}
	}
}
