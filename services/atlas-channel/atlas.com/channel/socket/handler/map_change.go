package handler

import (
	as "atlas-channel/account/session"
	"atlas-channel/cashshop"
	"atlas-channel/channel"
	"atlas-channel/character"
	"atlas-channel/portal"
	"atlas-channel/session"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const MapChangeHandle = "MapChangeHandle"

func MapChangeHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
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
			c, err := channel.NewProcessor(l, ctx).GetById(s.WorldId(), s.ChannelId())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve channel information being returned in to.")
				// TODO send server notice.
				return
			}

			err = cashshop.NewProcessor(l, ctx).Exit(s.CharacterId(), s.Map())
			if err != nil {
				l.WithError(err).Errorf("Unable to announce [%d] has returned from cash shop.", s.CharacterId())
			}

			err = as.NewProcessor(l, ctx).UpdateState(s.SessionId(), s.AccountId(), 2, model.ChannelChange{IPAddress: c.IpAddress(), Port: uint16(c.Port())})
			if err != nil {
				_ = session.NewProcessor(l, ctx).Destroy(s)
			}
			return
		}

		fieldKey = r.ReadByte()
		targetId = r.ReadUint32()
		portalName = r.ReadAsciiString()
		if len(portalName) == 0 {
			x = r.ReadInt16()
			y = r.ReadInt16()
		}
		unused = r.ReadByte()
		premium = r.ReadByte()
		if t.Region() == "GMS" && t.MajorVersion() >= 83 {
			chase = r.ReadBool()
		}
		if chase {
			targetX = r.ReadInt32()
			targetY = r.ReadInt32()
		}

		c, err := character.NewProcessor(l, ctx).GetById()(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to get character [%d]", s.CharacterId())
			return
		}
		if c.Hp() == 0 {
			l.Debugf("Character [%d] attempting to revive.", s.CharacterId())
			// TODO does the player own a wheel of fortune? 5510000
			// TODO if so, consume wheel cash item 5510000
			// TODO emit CharacterBattlefieldItemUseEffectBody if wheel item is used
			// TODO set hp to 50
			// TODO cancel all buffs

			// TODO if wheel of fortune was consumed, respawn in the same map, otherwise warp to return map of current map
			return
		}

		l.Debugf("Character [%d] attempting to enter portal [%s] at [%d,%d] heading to [%d]. FieldKey [%d].", s.CharacterId(), portalName, x, y, targetId, fieldKey)
		l.Debugf("Unused [%d], Premium [%d], Chase [%t], TargetX [%d], TargetY [%d]", unused, premium, chase, targetX, targetY)
		_ = portal.NewProcessor(l, ctx).Enter(s.Map(), portalName, s.CharacterId())
	}
}
