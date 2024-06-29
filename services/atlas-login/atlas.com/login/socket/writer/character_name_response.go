package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterNameResponse = "CharacterNameResponse"

type CharacterNameResponseCode string

const (
	CharacterNameResponseCodeOk                CharacterNameResponseCode = "OK"
	CharacterNameResponseCodeAlreadyRegistered CharacterNameResponseCode = "ALREADY_REGISTERED"
	CharacterNameResponseCodeNotAllowed        CharacterNameResponseCode = "NOT_ALLOWED"
	CharacterNameResponseCodeSystemError       CharacterNameResponseCode = "SYSTEM_ERROR"
)

func CharacterNameResponseBody(l logrus.FieldLogger) func(name string, code CharacterNameResponseCode) BodyProducer {
	return func(name string, code CharacterNameResponseCode) BodyProducer {
		return func(op uint16, options map[string]interface{}) []byte {
			w := response.NewWriter(l)
			w.WriteShort(op)
			w.WriteAsciiString(name)
			w.WriteByte(getCode(l)(code, options))
			rtn := w.Bytes()
			return rtn
		}
	}
}

const codeProperty = "codes"

func getCode(l logrus.FieldLogger) func(code CharacterNameResponseCode, options map[string]interface{}) byte {
	return func(codeString CharacterNameResponseCode, options map[string]interface{}) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options[codeProperty]; !ok {
			l.Errorf("Reason code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", codeString, CharacterNameResponse)
			return 99
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Reason code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", codeString, CharacterNameResponse)
			return 99
		}

		code, ok := codes[string(codeString)].(float64)
		if !ok {
			l.Errorf("Reason code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", codeString, CharacterNameResponse)
			return 99
		}
		return byte(code)
	}
}
