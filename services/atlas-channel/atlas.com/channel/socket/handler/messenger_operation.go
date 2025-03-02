package handler

import (
	"atlas-channel/character"
	"atlas-channel/invite"
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

func MessengerOperationHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
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
			tc, err := character.GetByName(l, ctx)(targetCharacter)
			if err != nil {
				l.WithError(err).Errorf("Unable to locate character by name [%s] to invite to messenger.", targetCharacter)
				return
			}

			os, err := session.GetByCharacterId(s.Tenant())(tc.Id())
			if err != nil || s.WorldId() != os.WorldId() || s.ChannelId() != os.ChannelId() {
				l.WithError(err).Errorf("Character [%d] not in channel. Cannot invite to messenger.", tc.Id())
				return
			}

			err = messenger.RequestInvite(l)(ctx)(s.CharacterId(), tc.Id())
			if err != nil {
				l.WithError(err).Errorf("Character [%d] was unable to request [%d] to join messenger.", s.CharacterId(), tc.Id())
			}
			return
		}
		if mode == MessengerOperationDeclineInvite {
			fromCharacter := r.ReadAsciiString()
			other := r.ReadAsciiString()
			zero := r.ReadByte()
			l.Debugf("Character [%d] rejected [%s] invite to messenger. Other [%s], Zero [%d]", s.CharacterId(), fromCharacter, other, zero)
			tc, err := character.GetByName(l, ctx)(other)
			if err != nil {
				l.WithError(err).Errorf("Unable to locate character by name [%s] to invite to messenger.", other)
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
			return
		}
	}
}
