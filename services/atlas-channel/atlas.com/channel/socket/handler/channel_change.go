package handler

import (
	as "atlas-channel/account/session"
	"atlas-channel/channel"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const ChannelChangeHandle = "ChannelChangeHandle"

func ChannelChangeHandleFunc(l logrus.FieldLogger, span opentracing.Span, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	channelChangeFunc := session.Announce(l)(wp)(writer.ChannelChange)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		channelId := r.ReadByte()
		updateTime := r.ReadUint32()
		l.Debugf("Character [%d] attempting to change to channel [%d]. update_time [%d].", s.CharacterId(), channelId, updateTime)

		// TODO verify alive
		// TODO verify not in mini dungeon

		c, err := channel.GetById(l, span, s.Tenant())(s.WorldId(), channelId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve channel information being logged in to.")
			// TODO send server notice.
			return
		}

		resp, err := as.UpdateState(l, span, s.Tenant())(s.SessionId(), s.AccountId(), 2)
		if err != nil || resp.Code != "OK" {
			l.WithError(err).Errorf("Unable to update session for character [%d] attempting to switch to channel.", s.CharacterId())
			session.Destroy(l, span, session.GetRegistry(), s.Tenant().Id)(s)
			return
		}

		err = channelChangeFunc(s, writer.ChannelChangeBody(c.IpAddress(), uint16(c.Port())))
		if err != nil {
			l.WithError(err).Errorf("Unable to write change channel.")
			return
		}
	}
}
