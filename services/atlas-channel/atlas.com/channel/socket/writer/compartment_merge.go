package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
)

const CompartmentMerge = "CompartmentMerge"

func CompartmentMergeBody(inventoryType byte) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(0)
		w.WriteByte(inventoryType)
		return w.Bytes()
	}
}