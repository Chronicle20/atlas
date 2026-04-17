package model

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type Buddy struct {
	FriendId    uint32
	FriendName  string
	Flag        byte
	ChannelId   channel.Id
	FriendGroup string
}

func (b Buddy) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
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
