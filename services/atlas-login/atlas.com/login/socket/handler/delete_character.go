package handler

import (
	"atlas-login/account"
	"atlas-login/character"
	"atlas-login/guild"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"
	"net"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const DeleteCharacterHandle = "DeleteCharacterHandle"

func DeleteCharacterHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		var verifyPic = false
		var pic string
		var dob uint32

		if t.Region() == "GMS" && t.MajorVersion() > 82 {
			verifyPic = true
			pic = r.ReadAsciiString()
		} else if t.Region() == "GMS" {
			dob = r.ReadUint32()
		}
		characterId := r.ReadUint32()
		l.Debugf("Handling delete character [%d] for account [%d]. verifyPic [%t]. verifyDob [%t]", characterId, s.AccountId(), verifyPic, dob != 0)

		if verifyPic {
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
				err = session.Announce(l)(ctx)(wp)(writer.DeleteCharacterResponse)(writer.DeleteCharacterErrorBody(characterId, writer.DeleteCharacterCodeUnknownError))(s)
				if err != nil {
					l.WithError(err).Errorf("Failed to write delete character response body.")
				}
				return
			}

			if a.PIC() != pic {
				l.Debugf("Failing character deletion due to PIC being incorrect.")
				_, limitReached, _ := ap.RecordPicAttempt(s.AccountId(), false, ipAddress, "")
				if limitReached {
					l.Warnf("Account [%d] has exceeded PIC attempt limit. Terminating session.", s.AccountId())
					_ = session.NewProcessor(l, ctx).Destroy(s)
					return
				}
				err = session.Announce(l)(ctx)(wp)(writer.DeleteCharacterResponse)(writer.DeleteCharacterErrorBody(characterId, writer.DeleteCharacterCodeSecondaryPinMismatch))(s)
				if err != nil {
					l.WithError(err).Errorf("Failed to write delete character response body.")
				}
				return
			}

			ap.RecordPicAttempt(s.AccountId(), true, ipAddress, "")
		}

		_, err := character.NewProcessor(l, ctx).GetById()(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] being deleted.", characterId)
			err = session.Announce(l)(ctx)(wp)(writer.DeleteCharacterResponse)(writer.DeleteCharacterErrorBody(characterId, writer.DeleteCharacterCodeUnknownError))(s)
			if err != nil {
				l.WithError(err).Errorf("Failed to write delete character response body.")
			}
			return
		}

		isGuildMaster, err := guild.NewProcessor(l, ctx).IsGuildMaster(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to check if character [%d] is a guild master.", characterId)
			err = session.Announce(l)(ctx)(wp)(writer.DeleteCharacterResponse)(writer.DeleteCharacterErrorBody(characterId, writer.DeleteCharacterCodeUnknownError))(s)
			if err != nil {
				l.WithError(err).Errorf("Failed to write delete character response body.")
			}
			return
		}

		if isGuildMaster {
			l.Debugf("Failing character deletion because character [%d] is a guild master.", characterId)
			err = session.Announce(l)(ctx)(wp)(writer.DeleteCharacterResponse)(writer.DeleteCharacterErrorBody(characterId, writer.DeleteCharacterCodeCannotDeleteGuildMaster))(s)
			if err != nil {
				l.WithError(err).Errorf("Failed to write delete character response body.")
			}
			return
		}

		// TODO - verify the character is not engaged.
		// TODO - verify the character is not part of a family.

		err = character.NewProcessor(l, ctx).DeleteById(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to delete character [%d].", characterId)
			err = session.Announce(l)(ctx)(wp)(writer.DeleteCharacterResponse)(writer.DeleteCharacterErrorBody(characterId, writer.DeleteCharacterCodeUnknownError))(s)
			if err != nil {
				l.WithError(err).Errorf("Failed to write delete character response body.")
			}
			return
		}

		err = session.Announce(l)(ctx)(wp)(writer.DeleteCharacterResponse)(writer.DeleteCharacterResponseBody(characterId))(s)
		if err != nil {
			l.WithError(err).Errorf("Failed to write delete character response body.")
		}
	}
}
