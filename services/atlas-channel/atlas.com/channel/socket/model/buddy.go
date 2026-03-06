package model

import (
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type Buddy struct {
	FriendId    uint32
	FriendName  string
	Flag        byte
	ChannelId   channel.Id
	FriendGroup string
}

func (b *Buddy) Encoder(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(b.FriendId)
		WritePaddedString(w, b.FriendName, 13)
		w.WriteByte(b.Flag)
		w.WriteInt32(int32(b.ChannelId))
		WritePaddedString(w, b.FriendGroup, 17)
		return w.Bytes()
	}
}

// TODO test with JMS before moving to library
func WritePaddedString(w *response.Writer, str string, number int) {
	if len(str) > number {
		w.WriteByteArray([]byte(str)[:number])
	} else {
		w.WriteByteArray([]byte(str))
		w.WriteByteArray(make([]byte, number-len(str)))
	}
}
