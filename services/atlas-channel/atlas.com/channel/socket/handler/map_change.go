package handler

import (
	as "atlas-channel/account/session"
	"atlas-channel/cashshop"
	"atlas-channel/channel"
	"atlas-channel/character"
	"atlas-channel/portal"
	"atlas-channel/respawn"
	"atlas-channel/session"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"

	_map "github.com/Chronicle20/atlas-constants/map"
	field2 "github.com/Chronicle20/atlas-packet/field"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func MapChangeHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := field2.Change{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		if p.CashShopReturn() {
			l.Debugf("Character [%d] returning from cash shop.", s.CharacterId())
			c, err := channel.NewProcessor(l, ctx).GetById(s.Field().Channel())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve channel information being returned in to.")
				return
			}

			err = cashshop.NewProcessor(l, ctx).Exit(s.CharacterId(), s.Field())
			if err != nil {
				l.WithError(err).Errorf("Unable to announce [%d] has returned from cash shop.", s.CharacterId())
			}

			err = as.NewProcessor(l, ctx).UpdateState(s.SessionId(), s.AccountId(), 2, model.ChannelChange{IPAddress: c.IpAddress(), Port: uint16(c.Port())})
			if err != nil {
				_ = session.NewProcessor(l, ctx).Destroy(s)
			}
			return
		}

		c, err := character.NewProcessor(l, ctx).GetById()(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to get character [%d]", s.CharacterId())
			return
		}
		if c.Hp() == 0 {
			l.Debugf("Character [%d] attempting to revive.", s.CharacterId())
			err = respawn.NewProcessor(l, ctx).Respawn(s.Field().Channel(), s.CharacterId(), s.MapId())
			if err != nil {
				l.WithError(err).Errorf("Unable to process respawn for character [%d].", s.CharacterId())
			}
			return
		}

		l.Debugf("Character [%d] attempting to enter portal [%s] at [%d,%d] heading to [%d]. FieldKey [%d].", s.CharacterId(), p.PortalName(), p.X(), p.Y(), p.TargetId(), p.FieldKey())
		if p.PortalName() == "" {
			_ = portal.NewProcessor(l, ctx).Warp(s.Field(), s.CharacterId(), _map.Id(p.TargetId()))
		} else {
			_ = portal.NewProcessor(l, ctx).Enter(s.Field(), p.PortalName(), s.CharacterId())
		}
	}
}
