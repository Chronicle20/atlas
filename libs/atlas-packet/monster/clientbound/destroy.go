package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MonsterDestroyWriter = "DestroyMonster"

type DestroyType byte

const (
	DestroyTypeDisappear DestroyType = 0
	DestroyTypeFadeOut   DestroyType = 1
	// DestroyTypeSwallow corresponds to v95 CMobPool::OnMobLeaveField's
	// destroyType == 4 path: the mob was swallowed by a character-eater
	// (e.g. Yeti-and-Pepe). The wire carries an additional `int32(swallowCharacterId)`
	// identifying the character that swallowed it.
	DestroyTypeSwallow DestroyType = 4
)

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
func NewMonsterDestroyBySwallow(uniqueId uint32, swallowCharacterId uint32) Destroy {
	return Destroy{
		uniqueId:           uniqueId,
		destroyType:        DestroyTypeSwallow,
		swallowCharacterId: swallowCharacterId,
	}
}

func (m Destroy) UniqueId() uint32                 { return m.uniqueId }
func (m Destroy) DestroyType() DestroyType         { return m.destroyType }
func (m Destroy) SwallowCharacterId() uint32       { return m.swallowCharacterId }
func (m Destroy) Operation() string                { return MonsterDestroyWriter }
func (m Destroy) String() string {
	return fmt.Sprintf("uniqueId [%d], destroyType [%d]", m.uniqueId, m.destroyType)
}

func (m Destroy) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.uniqueId)
		w.WriteByte(byte(m.destroyType))
		if m.destroyType == DestroyTypeSwallow {
			w.WriteInt(m.swallowCharacterId)
		}
		return w.Bytes()
	}
}

func (m *Destroy) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.uniqueId = r.ReadUint32()
		m.destroyType = DestroyType(r.ReadByte())
		if m.destroyType == DestroyTypeSwallow {
			m.swallowCharacterId = r.ReadUint32()
		}
	}
}
