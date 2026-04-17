package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterBuffCancelHandle = "CharacterBuffCancel"

// BuffCancelRequest - CUser::SendTemporaryStatResetRequest
type BuffCancelRequest struct {
	skillId int32
}

func (m BuffCancelRequest) SkillId() int32 { return m.skillId }

func (m BuffCancelRequest) Operation() string {
	return CharacterBuffCancelHandle
}

func (m BuffCancelRequest) String() string {
	return fmt.Sprintf("skillId [%d]", m.skillId)
}

func (m BuffCancelRequest) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(m.skillId)
		return w.Bytes()
	}
}

func (m *BuffCancelRequest) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.skillId = r.ReadInt32()
	}
}
