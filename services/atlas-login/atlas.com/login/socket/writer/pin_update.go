package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PinUpdate = "PinUpdate"

type PinUpdateMode string

const (
	PinUpdateModeOk    PinUpdateMode = "OK"
	PinUpdateModeError PinUpdateMode = "ERROR"
)

func PinUpdateBody(mode PinUpdateMode) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCode(l)(PinUpdate, string(mode), "modes", options))
			return w.Bytes()
		}
	}
}
