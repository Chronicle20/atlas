package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const KiteDestroyWriter = "DestroyKite"

// KiteDestroyAnimationType is the leading byte of REMOVE_KITE. It is a
// SUPPRESS-ANIMATION FLAG, not a selector between two animations.
// CMessageBoxPool::OnMessageBoxLeaveField (gms_v95 @0x635d60) always calls
// RemoveMessageBox, then gates the RelMove / canvas swap /
// CAnimationDisplayer::RegisterOneTimeAnimation despawn sequence on the byte
// being zero. Any non-zero value removes the banner instantly with no visual.
type KiteDestroyAnimationType byte

const (
	// KiteDestroyAnimated plays the one-shot despawn animation.
	KiteDestroyAnimated KiteDestroyAnimationType = 0
	// KiteDestroySilent removes the banner with no visual.
	KiteDestroySilent KiteDestroyAnimationType = 1
)

// packet-audit:fname CMessageBoxPool::OnMessageBoxLeaveField
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
