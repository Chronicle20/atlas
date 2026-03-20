package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterItemCancelHandle = "CharacterItemCancelHandle"

// ItemCancel - CUser::SendResetTemporaryStatRequest
type ItemCancel struct {
	sourceId int32
}

func (m ItemCancel) SourceId() int32 { return m.sourceId }

func (m ItemCancel) Operation() string {
	return CharacterItemCancelHandle
}

func (m ItemCancel) String() string {
	return fmt.Sprintf("sourceId [%d]", m.sourceId)
}

func (m ItemCancel) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(m.sourceId)
		return w.Bytes()
	}
}

func (m *ItemCancel) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.sourceId = r.ReadInt32()
	}
}
