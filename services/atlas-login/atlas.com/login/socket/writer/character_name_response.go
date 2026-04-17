package writer

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"

	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
)


type CharacterNameResponseCode string

const (
	CharacterNameResponseCodeOk                CharacterNameResponseCode = "OK"
	CharacterNameResponseCodeAlreadyRegistered CharacterNameResponseCode = "ALREADY_REGISTERED"
	CharacterNameResponseCodeNotAllowed        CharacterNameResponseCode = "NOT_ALLOWED"
	CharacterNameResponseCodeSystemError       CharacterNameResponseCode = "SYSTEM_ERROR"
)

func CharacterNameResponseBody(name string, code CharacterNameResponseCode) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			resolved := getCode(l)(charpkt.CharacterNameResponseWriter, string(code), "codes", options)
			return charpkt.NewCharacterNameResponse(name, resolved).Encode(l, ctx)(options)
		}
	}
}
