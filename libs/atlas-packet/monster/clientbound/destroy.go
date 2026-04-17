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
)

type Destroy struct {
	uniqueId    uint32
	destroyType DestroyType
}

func NewMonsterDestroy(uniqueId uint32, destroyType DestroyType) Destroy {
	return Destroy{uniqueId: uniqueId, destroyType: destroyType}
}

func (m Destroy) UniqueId() uint32      { return m.uniqueId }
func (m Destroy) DestroyType() DestroyType { return m.destroyType }
func (m Destroy) Operation() string     { return MonsterDestroyWriter }
func (m Destroy) String() string {
	return fmt.Sprintf("uniqueId [%d], destroyType [%d]", m.uniqueId, m.destroyType)
}

func (m Destroy) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.uniqueId)
		w.WriteByte(byte(m.destroyType))
		return w.Bytes()
	}
}

func (m *Destroy) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.uniqueId = r.ReadUint32()
		m.destroyType = DestroyType(r.ReadByte())
	}
}
