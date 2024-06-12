package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SelectWorld = "SelectWorld"

func SelectWorldBody(l logrus.FieldLogger) func(worldId int) BodyProducer {
	return func(worldId int) BodyProducer {
		return func(op uint16, options map[string]interface{}) []byte {
			//According to GMS, it should be the world that contains the most characters (most active)
			w := response.NewWriter(l)
			w.WriteShort(op)
			w.WriteInt(uint32(worldId))
			return w.Bytes()
		}
	}
}
