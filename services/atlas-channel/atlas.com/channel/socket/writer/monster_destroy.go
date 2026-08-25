package writer

import (
	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	monstercb "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// DestroyMonsterCode keys the DestroyMonster writer's tenant operations
// table (DOM-25). CMob::m_nDeadType does NOT render the same way on every
// client version -- v95 (0x658610) has a full six-arm switch, v87 (0x6b4c78)
// collapses it to {0,1,3} -> CMob::OnDie and {2,4,5} -> the action-12 bomb
// (task-253 design §2.2) -- so the byte is resolved through the operations
// table like every other operation code, for uniformity, rather than passed
// through verbatim.
type DestroyMonsterCode string

const (
	DestroyMonsterDisappear      DestroyMonsterCode = "DISAPPEAR"
	DestroyMonsterFadeOut        DestroyMonsterCode = "FADE_OUT"
	DestroyMonsterBomb           DestroyMonsterCode = "BOMB"
	DestroyMonsterDestructByMiss DestroyMonsterCode = "DESTRUCT_BY_MISS"
	DestroyMonsterSwallow        DestroyMonsterCode = "SWALLOW"
	DestroyMonsterSelfDestruct   DestroyMonsterCode = "SELF_DESTRUCT"
)

func DestroyMonsterBody(uniqueId uint32, key DestroyMonsterCode) packet.Encode {
	return atlas_packet.WithResolvedCode("operations", string(key), func(code byte) packet.Encoder {
		return monstercb.NewMonsterDestroy(uniqueId, monstercb.DestroyType(code))
	})
}
