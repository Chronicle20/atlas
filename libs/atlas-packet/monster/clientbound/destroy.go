package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const MonsterDestroyWriter = "DestroyMonster"

type DestroyType byte

// DestroyType is CMob::m_nDeadType as CMobPool::Update switches on it. The
// same byte does NOT render the same way on every version — v95 (0x658610)
// has a full six-arm switch, v87 (0x6b4c78) collapses it to
// {0,1,3} -> CMob::OnDie and {2,4,5} -> the action-12 bomb. The server passes
// the WZ selfDestruction.action byte through verbatim and never remaps it;
// per-version rendering is the client's business (task-253 design §2.2).
const (
	// DestroyTypeDisappear removes the mob with no animation. OnMobLeaveField
	// removes a type-0 mob immediately rather than queueing it.
	DestroyTypeDisappear DestroyType = 0
	// DestroyTypeFadeOut is the ordinary death: v95 CMob::OnDie @0x64e4b0.
	DestroyTypeFadeOut DestroyType = 1
	// DestroyTypeBomb is v95 CMob::OnBomb @0x650ec0.
	DestroyTypeBomb DestroyType = 2
	// DestroyTypeDestructByMiss is v95 CMob::OnDestructByMiss @0x64ea30
	// (SE_MOB_DIE, m_nOneTimeAction = 40, m_nSuspended = 2).
	DestroyTypeDestructByMiss DestroyType = 3
	// DestroyTypeSwallow is v95 CMob::OnSwallowed @0x641810, which consumes
	// m_dwSwallowCharacterID. It is the ONLY dead-type that carries a trailing
	// wire field, and only on the versions hasSwallowCharacterId reports.
	DestroyTypeSwallow DestroyType = 4
	// DestroyTypeSelfDestruct renders as an ordinary die on v95 and as the
	// action-12 bomb on v87 (sub_69E44A).
	DestroyTypeSelfDestruct DestroyType = 5
)

// hasSwallowCharacterId reports whether CMobPool::OnMobLeaveField reads the
// trailing int32 that follows dead-type 4. Swept across all ten IDBs
// (task-253 design §2.1): absent on GMS v48 0x55957b, v61 0x5d4b87,
// v72 0x6258a1, v79 0x646ff6, v83 0x67961d, v84 0x6901b3, v87 0x6b5169;
// present on GMS v92 0x64bb90, v95 0x658b90 and JMS v185 0x6f8a1f. The JMS
// arm is derived independently, not left to fall out of MajorAtLeast(185 >= 92).
func hasSwallowCharacterId(t tenant.Model) bool {
	return (t.IsRegion("GMS") && t.MajorAtLeast(92)) || t.Region() == "JMS"
}

// packet-audit:fname CMobPool::OnMobLeaveField
type Destroy struct {
	uniqueId           uint32
	destroyType        DestroyType
	swallowCharacterId uint32
}

func NewMonsterDestroy(uniqueId uint32, destroyType DestroyType) Destroy {
	return Destroy{uniqueId: uniqueId, destroyType: destroyType}
}

// NewMonsterDestroyBySwallow emits the destroyType=4 wire shape with the
// trailing swallowCharacterId. Used when a character-eater mob consumes a
// player; the client renders the swallow animation against that character.
// The trailing field is version-gated to GMS >= 92 / JMS; see
// hasSwallowCharacterId.
func NewMonsterDestroyBySwallow(uniqueId uint32, swallowCharacterId uint32) Destroy {
	return Destroy{
		uniqueId:           uniqueId,
		destroyType:        DestroyTypeSwallow,
		swallowCharacterId: swallowCharacterId,
	}
}

func (m Destroy) UniqueId() uint32           { return m.uniqueId }
func (m Destroy) DestroyType() DestroyType   { return m.destroyType }
func (m Destroy) SwallowCharacterId() uint32 { return m.swallowCharacterId }
func (m Destroy) Operation() string          { return MonsterDestroyWriter }
func (m Destroy) String() string {
	return fmt.Sprintf("uniqueId [%d], destroyType [%d]", m.uniqueId, m.destroyType)
}

func (m Destroy) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.uniqueId)
		w.WriteByte(byte(m.destroyType))
		if m.destroyType == DestroyTypeSwallow && hasSwallowCharacterId(t) {
			w.WriteInt(m.swallowCharacterId)
		}
		return w.Bytes()
	}
}

func (m *Destroy) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.uniqueId = r.ReadUint32()
		m.destroyType = DestroyType(r.ReadByte())
		if m.destroyType == DestroyTypeSwallow && hasSwallowCharacterId(t) {
			m.swallowCharacterId = r.ReadUint32()
		}
	}
}
