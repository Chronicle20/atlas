package writer

import (
	"context"

	npcpkt "github.com/Chronicle20/atlas-packet/npc"
	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

const (
	GuideTalk = "GuideTalk"
)

func GuideTalkMessageBody(message string, width uint32, duration uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return npcpkt.NewGuideTalkMessage(message, width, duration).Encode(l, ctx)
	}
}

func GuideTalkIdxBody(hintId uint32, duration uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return npcpkt.NewGuideTalkIdx(hintId, duration).Encode(l, ctx)
	}
}
