package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const BuddyErrorWriter = "BuddyError"

type Error struct {
	mode     byte
	hasExtra bool
}

func NewBuddyError(mode byte, hasExtra bool) Error {
	return Error{mode: mode, hasExtra: hasExtra}
}

func (m Error) Mode() byte        { return m.mode }
func (m Error) HasExtra() bool     { return m.hasExtra }
func (m Error) Operation() string  { return BuddyErrorWriter }

func (m Error) String() string {
	return fmt.Sprintf("buddy error mode [%d]", m.mode)
}

func (m Error) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		if m.hasExtra {
			w.WriteByte(0)
		}
		return w.Bytes()
	}
}

func (m *Error) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		if m.hasExtra {
			_ = r.ReadByte()
		}
	}
}
