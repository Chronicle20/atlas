package writer

import (
	"atlas-login/character"
	"atlas-login/tenant"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const AddCharacterEntry = "AddCharacterEntry"

func AddCharacterEntryBody(l logrus.FieldLogger, tenant tenant.Model) func(c character.Model) BodyProducer {
	return func(c character.Model) BodyProducer {
		return func(op uint16, options map[string]interface{}) []byte {
			w := response.NewWriter(l)
			w.WriteShort(op)
			w.WriteByte(getAddCharacterCode(l)(AddCharacterCodeOk, options))
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
		return func(op uint16, options map[string]interface{}) []byte {
			w := response.NewWriter(l)
			w.WriteShort(op)
			w.WriteByte(getAddCharacterCode(l)(code, options))
			return w.Bytes()
		}
	}
}

const addCharacterCodeProperty = "codes"

func getAddCharacterCode(l logrus.FieldLogger) func(code AddCharacterCode, options map[string]interface{}) byte {
	return func(codeString AddCharacterCode, options map[string]interface{}) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options[addCharacterCodeProperty]; !ok {
			l.Errorf("Reason code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", codeString, AddCharacterEntry)
			return 99
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Reason code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", codeString, AddCharacterEntry)
			return 99
		}

		code, ok := codes[string(codeString)].(float64)
		if !ok {
			l.Errorf("Reason code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", codeString, AddCharacterEntry)
			return 99
		}
		return byte(code)
	}
}
