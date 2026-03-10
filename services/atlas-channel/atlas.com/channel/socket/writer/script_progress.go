package writer

import (
	"context"

	questpkt "github.com/Chronicle20/atlas-packet/quest"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const (
	ScriptProgress = "ScriptProgress"
)

func ScriptProgressBody(message string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return questpkt.NewScriptProgress(message).Encode(l, ctx)
	}
}
