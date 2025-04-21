package handler

import (
	"atlas-channel/character"
	"atlas-channel/invite"
	"atlas-channel/message"
	"atlas-channel/messenger"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

type MessengerOperation byte

const (
	MessengerOperationHandle       = "MessengerOperationHandle"
	MessengerOperationAnswerInvite = MessengerOperation(0)
	//MessengerOperationCreate        = MessengerOperation(1)
	MessengerOperationClose         = MessengerOperation(2)
	MessengerOperationInvite        = MessengerOperation(3)
	MessengerOperationDeclineInvite = MessengerOperation(5)
	MessengerOperationChat          = MessengerOperation(6)
)

func MessengerOperationHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		mode := MessengerOperation(r.ReadByte())
		if mode == MessengerOperationAnswerInvite {
			messengerId := r.ReadUint32()
			l.Debugf("Character [%d] answered messenger [%d] invite.", s.CharacterId(), messengerId)
			if messengerId == 0 {
				err := messenger.Create(l)(ctx)(s.CharacterId())
				if err != nil {
					l.WithError(err).Errorf("Unable to issue create messenger for character [%d].", s.CharacterId())
				}
			} else {
				err := invite.Accept(l)(ctx)(s.CharacterId(), s.WorldId(), invite.InviteTypeMessenger, messengerId)
				if err != nil {
					l.WithError(err).Errorf("Unable to issue invite acceptance command for character [%d].", s.CharacterId())
				}
			}
			return
		}
		if mode == MessengerOperationClose {
			l.Debugf("Character [%d] exited messenger.", s.CharacterId())
			m, err := messenger.GetByMemberId(l)(ctx)(s.CharacterId())
			if err != nil {
				return
			}
			err = messenger.Leave(l)(ctx)(m.Id(), s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Unable to issue create messenger for character [%d].", s.CharacterId())
			}
			return
		}
		if mode == MessengerOperationInvite {
			targetCharacter := r.ReadAsciiString()
			l.Debugf("Character [%d] attempting to invite [%s] to messenger.", s.CharacterId(), targetCharacter)
			tc, err := character.NewProcessor(l, ctx).GetByName(targetCharacter)
			if err != nil {
				l.WithError(err).Errorf("Unable to locate character by name [%s] to invite to messenger.", targetCharacter)
				err = session.Announce(l)(ctx)(wp)(writer.MessengerOperation)(writer.MessengerOperationInviteSentBody(targetCharacter, false))(s)
				if err != nil {
					l.WithError(err).Errorf("Character [%d] was unable to request [%d] to invite messenger.", s.CharacterId(), tc.Id())
				}
				return
			}

			err = messenger.RequestInvite(l)(ctx)(s.CharacterId(), tc.Id())
			if err != nil {
				l.WithError(err).Errorf("Character [%d] was unable to request [%d] to invite messenger.", s.CharacterId(), tc.Id())
			}

			err = session.Announce(l)(ctx)(wp)(writer.MessengerOperation)(writer.MessengerOperationInviteSentBody(targetCharacter, true))(s)
			if err != nil {
				l.WithError(err).Errorf("Character [%d] was unable to request [%d] to invite messenger.", s.CharacterId(), tc.Id())
			}
			return
		}
		if mode == MessengerOperationDeclineInvite {
			fromName := r.ReadAsciiString()
			myName := r.ReadAsciiString()
			alwaysZero := r.ReadByte()
			l.Debugf("Character [%d] rejected [%s] invite to messenger. Other [%s], Zero [%d]", s.CharacterId(), fromName, myName, alwaysZero)
			tc, err := character.NewProcessor(l, ctx).GetByName(fromName)
			if err != nil {
				l.WithError(err).Errorf("Unable to locate character by name [%s] to reject invitation of.", fromName)
				return
			}
			err = invite.Reject(l)(ctx)(s.CharacterId(), s.WorldId(), invite.InviteTypeMessenger, tc.Id())
			if err != nil {
				l.WithError(err).Errorf("Unable to issue invite rejection command for character [%d].", s.CharacterId())
			}
			return
		}
		if mode == MessengerOperationChat {
			msg := r.ReadAsciiString()
			l.Debugf("Character [%d] sending message [%s] to messenger.", s.CharacterId(), msg)
			m, err := messenger.GetByMemberId(l)(ctx)(s.CharacterId())
			if err != nil {
				return
			}
			rids := make([]uint32, 0)
			for _, mm := range m.Members() {
				if mm.Id() != s.CharacterId() {
					rids = append(rids, mm.Id())
				}
			}
			err = message.MessengerChat(l)(ctx)(s.Map(), s.CharacterId(), msg, rids)
			if err != nil {
				l.WithError(err).Errorf("Unable to relay messenger [%d] to recipients.", m.Id())
			}
			return
		}
	}
}
