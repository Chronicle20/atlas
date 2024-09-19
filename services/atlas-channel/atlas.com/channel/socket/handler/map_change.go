package handler

import (
	as "atlas-channel/account/session"
	"atlas-channel/channel"
	"atlas-channel/kafka/producer"
	"atlas-channel/portal"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const MapChangeHandle = "MapChangeHandle"

func MapChangeHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	channelChangeFunc := session.Announce(l)(ctx)(wp)(writer.ChannelChange)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		cs := r.Available() == 0
		var fieldKey byte
		var targetId uint32
		var portalName string
		var x int16
		var y int16
		var unused byte
		var premium byte
		var chase bool
		var targetX int32
		var targetY int32

		if cs {
			l.Debugf("Character [%d] returning from cash shop.", s.CharacterId())
			c, err := channel.GetById(l)(ctx)(s.WorldId(), s.ChannelId())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve channel information being returned in to.")
				// TODO send server notice.
				return
			}

			resp, err := as.UpdateState(l)(ctx)(s.SessionId(), s.AccountId(), 2)
			if err != nil || resp.Code != "OK" {
				l.WithError(err).Errorf("Unable to update session for character [%d] attempting to switch to channel.", s.CharacterId())
				session.Destroy(l, ctx, session.GetRegistry())(s)
				return
			}

			err = channelChangeFunc(s, writer.ChannelChangeBody(c.IpAddress(), uint16(c.Port())))
			if err != nil {
				l.WithError(err).Errorf("Unable to write change channel.")
				return
			}
			return
		}

		fieldKey = r.ReadByte()
		targetId = r.ReadUint32()
		portalName = r.ReadAsciiString()
		x = r.ReadInt16()
		y = r.ReadInt16()
		unused = r.ReadByte()
		premium = r.ReadByte()
		if t.Region() == "GMS" && t.MajorVersion() >= 83 {
			chase = r.ReadBool()
		}
		if chase {
			targetX = r.ReadInt32()
			targetY = r.ReadInt32()
		}

		l.Debugf("Character [%d] attempting to enter portal [%s] at [%d,%d] heading to [%d]. FieldKey [%d].", s.CharacterId(), portalName, x, y, targetId, fieldKey)
		l.Debugf("Unused [%d], Premium [%d], Chase [%t], TargetX [%d], TargetY [%d]", unused, premium, chase, targetX, targetY)
		_ = portal.Enter(l, ctx, producer.ProviderImpl(l)(ctx))(s.WorldId(), s.ChannelId(), s.MapId(), portalName, s.CharacterId())
	}
}
