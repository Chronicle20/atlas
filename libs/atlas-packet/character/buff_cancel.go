package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterBuffCancelHandle = "CharacterBuffCancel"

// BuffCancel - CUser::SendTemporaryStatResetRequest
type BuffCancel struct {
	skillId int32
}

func (m BuffCancel) SkillId() int32 { return m.skillId }

func (m BuffCancel) Operation() string {
	return CharacterBuffCancelHandle
}

func (m BuffCancel) String() string {
	return fmt.Sprintf("skillId [%d]", m.skillId)
}

func (m BuffCancel) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(m.skillId)
		return w.Bytes()
	}
}

func (m *BuffCancel) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.skillId = r.ReadInt32()
	}
}
