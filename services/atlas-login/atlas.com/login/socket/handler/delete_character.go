package handler

import (
	"atlas-login/account"
	"atlas-login/character"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const DeleteCharacterHandle = "DeleteCharacterHandle"

func DeleteCharacterHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader) {
	deleteCharacterResponseFunc := session.Announce(l)(wp)(writer.DeleteCharacterResponse)
	return func(s session.Model, r *request.Reader) {
		var verifyPic = false
		var pic string
		var dob uint32

		if s.Tenant().Region == "GMS" && s.Tenant().MajorVersion > 82 {
			verifyPic = true
			pic = r.ReadAsciiString()
		} else if s.Tenant().Region == "GMS" {
			dob = r.ReadUint32()
		}
		characterId := r.ReadUint32()
		l.Debugf("Handling delete character [%d] for account [%d]. verifyPic [%t] pic [%s]. verifyDob [%t] dob [%d]", characterId, s.AccountId(), verifyPic, pic, dob != 0, dob)

		if verifyPic {
			a, err := account.GetById(l, ctx, s.Tenant())(s.AccountId())
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

		_, err := character.GetById(l, ctx, s.Tenant())(characterId)
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

		err = character.DeleteById(l, ctx, s.Tenant())(characterId)
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
