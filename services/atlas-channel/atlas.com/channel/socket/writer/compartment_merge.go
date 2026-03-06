package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CompartmentMerge = "CompartmentMerge"

func CompartmentMergeBody(inventoryType byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(0)
			w.WriteByte(inventoryType)
			return w.Bytes()
		}
	}
}
