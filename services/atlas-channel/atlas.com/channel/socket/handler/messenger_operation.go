package handler

import (
	"atlas-channel/character"
	"atlas-channel/invite"
	"atlas-channel/message"
	"atlas-channel/messenger"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	invite2 "github.com/Chronicle20/atlas-constants/invite"
	messenger2 "github.com/Chronicle20/atlas-packet/messenger"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

type MessengerOperation byte

const (
	MessengerOperationAnswerInvite  = "ANSWER_INVITE"
	MessengerOperationCreate        = "CREATE"
	MessengerOperationClose         = "CLOSE"
	MessengerOperationInvite        = "INVITE"
	MessengerOperationDeclineInvite = "DECLINE_INVITE"
	MessengerOperationChat          = "CHAT"
)

func MessengerOperationHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := messenger2.Operation{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		mode := MessengerOperation(p.Mode())
		if isMessengerShopOperation(l)(readerOptions, mode, MessengerOperationAnswerInvite) {
			sp := &messenger2.OperationAnswerInvite{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] answered messenger [%d] invite.", s.CharacterId(), sp.MessengerId())
			if sp.MessengerId() == 0 {
				err := messenger.NewProcessor(l, ctx).Create(s.CharacterId())
				if err != nil {
					l.WithError(err).Errorf("Unable to issue create messenger for character [%d].", s.CharacterId())
				}
			} else {
				err := invite.NewProcessor(l, ctx).Accept(s.CharacterId(), s.WorldId(), string(invite2.TypeMessenger), sp.MessengerId())
				if err != nil {
					l.WithError(err).Errorf("Unable to issue invite acceptance command for character [%d].", s.CharacterId())
				}
			}
			return
		}
		if isMessengerShopOperation(l)(readerOptions, mode, MessengerOperationClose) {
			l.Debugf("Character [%d] exited messenger.", s.CharacterId())
			m, err := messenger.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())
			if err != nil {
				return
			}
			err = messenger.NewProcessor(l, ctx).Leave(m.Id(), s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to issue create messenger for character [%d].", s.CharacterId())
			}
			return
		}
		if isMessengerShopOperation(l)(readerOptions, mode, MessengerOperationInvite) {
			sp := &messenger2.OperationInvite{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] attempting to invite [%s] to messenger.", s.CharacterId(), sp.TargetCharacter())
			tc, err := character.NewProcessor(l, ctx).GetByName(sp.TargetCharacter())
			if err != nil {
				l.WithError(err).Errorf("Unable to locate character by name [%s] to invite to messenger.", sp.TargetCharacter())
				err = session.Announce(l)(ctx)(wp)(messenger2.MessengerOperationWriter)(messenger2.MessengerOperationInviteSentBody(sp.TargetCharacter(), false))(s)
				if err != nil {
					l.WithError(err).Errorf("Character [%d] was unable to request [%d] to invite messenger.", s.CharacterId(), tc.Id())
				}
				return
			}

			err = messenger.NewProcessor(l, ctx).RequestInvite(s.CharacterId(), tc.Id())
			if err != nil {
				l.WithError(err).Errorf("Character [%d] was unable to request [%d] to invite messenger.", s.CharacterId(), tc.Id())
			}

			err = session.Announce(l)(ctx)(wp)(messenger2.MessengerOperationWriter)(messenger2.MessengerOperationInviteSentBody(sp.TargetCharacter(), true))(s)
			if err != nil {
				l.WithError(err).Errorf("Character [%d] was unable to request [%d] to invite messenger.", s.CharacterId(), tc.Id())
			}
			return
		}
		if isMessengerShopOperation(l)(readerOptions, mode, MessengerOperationDeclineInvite) {
			sp := &messenger2.OperationDeclineInvite{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] rejected [%s] invite to messenger. Other [%s], Zero [%d]", s.CharacterId(), sp.FromName(), sp.MyName(), sp.AlwaysZero())
			tc, err := character.NewProcessor(l, ctx).GetByName(sp.FromName())
			if err != nil {
				l.WithError(err).Errorf("Unable to locate character by name [%s] to reject invitation of.", sp.FromName())
				return
			}
			err = invite.NewProcessor(l, ctx).Reject(s.CharacterId(), s.WorldId(), string(invite2.TypeMessenger), tc.Id())
			if err != nil {
				l.WithError(err).Errorf("Unable to issue invite rejection command for character [%d].", s.CharacterId())
			}
			return
		}
		if isMessengerShopOperation(l)(readerOptions, mode, MessengerOperationChat) {
			sp := &messenger2.OperationChat{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] sending message [%s] to messenger.", s.CharacterId(), sp.Msg())
			m, err := messenger.NewProcessor(l, ctx).GetByMemberId(s.CharacterId())
			if err != nil {
				return
			}
			rids := make([]uint32, 0)
			for _, mm := range m.Members() {
				if mm.Id() != s.CharacterId() {
					rids = append(rids, mm.Id())
				}
			}
			err = message.NewProcessor(l, ctx).MessengerChat(s.Field(), s.CharacterId(), sp.Msg(), rids)
			if err != nil {
				l.WithError(err).Errorf("Unable to relay messenger [%d] to recipients.", m.Id())
			}
			return
		}
	}
}

func isMessengerShopOperation(l logrus.FieldLogger) func(options map[string]interface{}, op MessengerOperation, key string) bool {
	return func(options map[string]interface{}, op MessengerOperation, key string) bool {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["operations"]; !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}

		res, ok := codes[key].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}
		return MessengerOperation(res) == op
	}
}
