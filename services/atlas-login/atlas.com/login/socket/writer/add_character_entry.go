package writer

import (
	"atlas-login/character"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const AddCharacterEntry = "AddCharacterEntry"

func AddCharacterEntryBody(l logrus.FieldLogger, tenant tenant.Model) func(c character.Model) BodyProducer {
	return func(c character.Model) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCode(l)(AddCharacterEntry, string(AddCharacterCodeOk), "codes", options))
			WriteCharacter(tenant)(w, c, false)
			return w.Bytes()
		}
	}
}

type AddCharacterCode string

const (
	AddCharacterCodeOk                       AddCharacterCode = "OK"
	AddCharacterCodeTooManyConnections       AddCharacterCode = "TOO_MANY_CONNECTIONS"
	AddCharacterCodeAccountRequestedTransfer AddCharacterCode = "ACCOUNT_REQUESTED_TRANSFER"
	AddCharacterCodeCannotUseName            AddCharacterCode = "CANNOT_USE_NAME"
	AddCharacterCodeUnknownError             AddCharacterCode = "UNKNOWN_ERROR"
)

func AddCharacterErrorBody(l logrus.FieldLogger, _ tenant.Model) func(code AddCharacterCode) BodyProducer {
	return func(code AddCharacterCode) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteByte(getCode(l)(AddCharacterEntry, string(code), "codes", options))
			return w.Bytes()
		}
	}
}
