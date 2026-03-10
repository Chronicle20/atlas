package field

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const KiteDestroyWriter = "KiteDestroy"

type KiteDestroyAnimationType byte

const (
	KiteDestroyAnimationType1 KiteDestroyAnimationType = 0
	KiteDestroyAnimationType2 KiteDestroyAnimationType = 1
)

type KiteDestroy struct {
	animationType KiteDestroyAnimationType
	id            uint32
}

func NewKiteDestroy(id uint32, animationType KiteDestroyAnimationType) KiteDestroy {
	return KiteDestroy{id: id, animationType: animationType}
}

func (m KiteDestroy) Operation() string { return KiteDestroyWriter }
func (m KiteDestroy) String() string {
	return fmt.Sprintf("id [%d], animationType [%d]", m.id, m.animationType)
}

func (m KiteDestroy) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(m.animationType))
		w.WriteInt(m.id)
		return w.Bytes()
	}
}

func (m *KiteDestroy) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.animationType = KiteDestroyAnimationType(r.ReadByte())
		m.id = r.ReadUint32()
	}
}
