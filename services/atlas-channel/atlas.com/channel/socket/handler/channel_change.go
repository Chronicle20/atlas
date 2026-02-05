package handler

import (
	as "atlas-channel/account/session"
	"atlas-channel/channel"
	"atlas-channel/session"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"

	channel2 "github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const ChannelChangeHandle = "ChannelChangeHandle"

func ChannelChangeHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		channelId := channel2.Id(r.ReadByte())
		updateTime := r.ReadUint32()
		l.Debugf("Character [%d] attempting to change to channel [%d]. update_time [%d].", s.CharacterId(), channelId, updateTime)

		// TODO verify alive
		// TODO verify not in mini dungeon

		c, err := channel.NewProcessor(l, ctx).GetById(channel2.NewModel(s.WorldId(), channelId))
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve channel information being logged in to.")
			// TODO send server notice.
			return
		}

		err = as.NewProcessor(l, ctx).UpdateState(s.SessionId(), s.AccountId(), 2, model.ChannelChange{IPAddress: c.IpAddress(), Port: uint16(c.Port())})
		if err != nil {
			_ = session.NewProcessor(l, ctx).Destroy(s)
		}
	}
}
