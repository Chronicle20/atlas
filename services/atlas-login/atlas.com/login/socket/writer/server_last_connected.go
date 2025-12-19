package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
)

const SelectWorld = "SelectWorld"

func SelectWorldBody(worldId int) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		//According to GMS, it should be the world that contains the most characters (most active)
		w.WriteInt(uint32(worldId))
		return w.Bytes()
	}
}
