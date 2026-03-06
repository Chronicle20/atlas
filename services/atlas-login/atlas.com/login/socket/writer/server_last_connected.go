package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SelectWorld = "SelectWorld"

func SelectWorldBody(worldId int) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			//According to GMS, it should be the world that contains the most characters (most active)
			w.WriteInt(uint32(worldId))
			return w.Bytes()
		}
	}
}
