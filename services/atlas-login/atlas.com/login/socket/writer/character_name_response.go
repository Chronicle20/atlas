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
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteAsciiString(name)
			w.WriteByte(getCode(l)(CharacterNameResponse, string(code), "codes", options))
			rtn := w.Bytes()
			return rtn
		}
	}
}
