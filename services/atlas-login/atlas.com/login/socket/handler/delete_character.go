package handler

import (
	"atlas-login/account"
	"atlas-login/character"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const DeleteCharacterHandle = "DeleteCharacterHandle"

func DeleteCharacterHandleFunc(l logrus.FieldLogger, span opentracing.Span, wp writer.Producer) func(s session.Model, r *request.Reader) {
	deleteCharacterResponseFunc := session.Announce(wp)(writer.DeleteCharacterResponse)
	return func(s session.Model, r *request.Reader) {
		var verifyPic = false
		var pic string

		if s.Tenant().Region == "GMS" {
			verifyPic = true
			pic = r.ReadAsciiString()
		}
		characterId := r.ReadUint32()
		l.Debugf("Handling delete character [%d] for account [%d]. verifyPic [%t] pic [%s].", characterId, s.AccountId(), verifyPic, pic)

		if verifyPic {
			a, err := account.GetById(l, span, s.Tenant())(s.AccountId())
			if err != nil {
				l.WithError(err).Errorf("Unable to retrieve account performing deletion.")
				err = deleteCharacterResponseFunc(s, writer.DeleteCharacterErrorBody(l, s.Tenant())(characterId, writer.DeleteCharacterCodeUnknownError))
				if err != nil {
					l.WithError(err).Errorf("Failed to write delete character response body.")
				}
				return
			}

			if a.PIC() != pic {
				l.Debugf("Failing character deletion due to PIC being incorrect.")
				err = deleteCharacterResponseFunc(s, writer.DeleteCharacterErrorBody(l, s.Tenant())(characterId, writer.DeleteCharacterCodeSecondaryPinMismatch))
				if err != nil {
					l.WithError(err).Errorf("Failed to write delete character response body.")
				}
				return
			}
		}

		_, err := character.GetById(l, span, s.Tenant())(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] being deleted.", characterId)
			err = deleteCharacterResponseFunc(s, writer.DeleteCharacterErrorBody(l, s.Tenant())(characterId, writer.DeleteCharacterCodeUnknownError))
			if err != nil {
				l.WithError(err).Errorf("Failed to write delete character response body.")
			}
			return
		}

		// TODO - verify the character is not a guild master.
		// TODO - verify the character is not engaged.
		// TODO - verify the character is not part of a family.

		err = character.DeleteById(l, span, s.Tenant())(characterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to delete character [%d].", characterId)
			err = deleteCharacterResponseFunc(s, writer.DeleteCharacterErrorBody(l, s.Tenant())(characterId, writer.DeleteCharacterCodeUnknownError))
			if err != nil {
				l.WithError(err).Errorf("Failed to write delete character response body.")
			}
			return
		}

		err = deleteCharacterResponseFunc(s, writer.DeleteCharacterResponseBody(l, s.Tenant())(characterId))
		if err != nil {
			l.WithError(err).Errorf("Failed to write delete character response body.")
		}
	}
}
