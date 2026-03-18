package handler

import (
	"atlas-login/account"
	"atlas-login/character"
	"atlas-login/guild"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"
	"net"

	charcb "github.com/Chronicle20/atlas-packet/character/clientbound"
	charsb "github.com/Chronicle20/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func DeleteCharacterHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := charsb.DeleteCharacter{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		if p.VerifyPic() {
			ap := account.NewProcessor(l, ctx)
			ipAddress := ""
			if addr := s.GetRemoteAddress(); addr != nil {
				if tcpAddr, ok := addr.(*net.TCPAddr); ok {
					ipAddress = tcpAddr.IP.String()
				} else {
					host, _, err := net.SplitHostPort(addr.String())
					if err == nil {
						ipAddress = host
					}
				}
			}

			a, err := ap.GetById(s.AccountId())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve account performing deletion.")
				err = session.Announce(l)(ctx)(wp)(charcb.DeleteCharacterResponseWriter)(writer.DeleteCharacterErrorBody(p.CharacterId(), writer.DeleteCharacterCodeUnknownError))(s)
				if err != nil {
					l.WithError(err).Errorf("Failed to write delete character response body.")
				}
				return
			}

			if a.PIC() != p.Pic() {
				l.Debugf("Failing character deletion due to PIC being incorrect.")
				_, limitReached, _ := ap.RecordPicAttempt(s.AccountId(), false, ipAddress, "")
				if limitReached {
					l.Warnf("Account [%d] has exceeded PIC attempt limit. Terminating session.", s.AccountId())
					_ = session.NewProcessor(l, ctx).Destroy(s)
					return
				}
				err = session.Announce(l)(ctx)(wp)(charcb.DeleteCharacterResponseWriter)(writer.DeleteCharacterErrorBody(p.CharacterId(), writer.DeleteCharacterCodeSecondaryPinMismatch))(s)
				if err != nil {
					l.WithError(err).Errorf("Failed to write delete character response body.")
				}
				return
			}

			ap.RecordPicAttempt(s.AccountId(), true, ipAddress, "")
		}

		_, err := character.NewProcessor(l, ctx).GetById()(p.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] being deleted.", p.CharacterId())
			err = session.Announce(l)(ctx)(wp)(charcb.DeleteCharacterResponseWriter)(writer.DeleteCharacterErrorBody(p.CharacterId(), writer.DeleteCharacterCodeUnknownError))(s)
			if err != nil {
				l.WithError(err).Errorf("Failed to write delete character response body.")
			}
			return
		}

		isGuildMaster, err := guild.NewProcessor(l, ctx).IsGuildMaster(p.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to check if character [%d] is a guild master.", p.CharacterId())
			err = session.Announce(l)(ctx)(wp)(charcb.DeleteCharacterResponseWriter)(writer.DeleteCharacterErrorBody(p.CharacterId(), writer.DeleteCharacterCodeUnknownError))(s)
			if err != nil {
				l.WithError(err).Errorf("Failed to write delete character response body.")
			}
			return
		}

		if isGuildMaster {
			l.Debugf("Failing character deletion because character [%d] is a guild master.", p.CharacterId())
			err = session.Announce(l)(ctx)(wp)(charcb.DeleteCharacterResponseWriter)(writer.DeleteCharacterErrorBody(p.CharacterId(), writer.DeleteCharacterCodeCannotDeleteGuildMaster))(s)
			if err != nil {
				l.WithError(err).Errorf("Failed to write delete character response body.")
			}
			return
		}

		// TODO - verify the character is not engaged.
		// TODO - verify the character is not part of a family.

		err = character.NewProcessor(l, ctx).DeleteById(p.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to delete character [%d].", p.CharacterId())
			err = session.Announce(l)(ctx)(wp)(charcb.DeleteCharacterResponseWriter)(writer.DeleteCharacterErrorBody(p.CharacterId(), writer.DeleteCharacterCodeUnknownError))(s)
			if err != nil {
				l.WithError(err).Errorf("Failed to write delete character response body.")
			}
			return
		}

		err = session.Announce(l)(ctx)(wp)(charcb.DeleteCharacterResponseWriter)(writer.DeleteCharacterResponseBody(p.CharacterId()))(s)
		if err != nil {
			l.WithError(err).Errorf("Failed to write delete character response body.")
		}
	}
}
