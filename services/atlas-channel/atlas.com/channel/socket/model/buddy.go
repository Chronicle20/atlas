package model

import (
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Buddy struct {
	FriendId    uint32
	FriendName  string
	Flag        byte
	ChannelId   channel.Id
	FriendGroup string
}

func (b *Buddy) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt(b.FriendId)
		WritePaddedString(w, b.FriendName, 13)
		w.WriteByte(b.Flag)
		w.WriteInt32(int32(b.ChannelId))
		WritePaddedString(w, b.FriendGroup, 17)
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
