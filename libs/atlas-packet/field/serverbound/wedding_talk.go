package serverbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const WeddingTalkHandle = "WeddingTalk"

// WeddingTalk - CField_Wedding::OnWeddingProgress#Talk
// Emitted on the bless YESNO confirm (witness path). Empty body (header only).
type WeddingTalk struct {
}

func NewWeddingTalk() WeddingTalk {
	return WeddingTalk{}
}

func (m WeddingTalk) Operation() string {
	return WeddingTalkHandle
}

func (m WeddingTalk) String() string {
	return "empty"
}

func (m WeddingTalk) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

func (m *WeddingTalk) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
