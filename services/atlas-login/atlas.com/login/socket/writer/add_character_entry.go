package writer

import (
	"atlas-login/character"
	"context"

	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)


func AddCharacterEntryBody(c character.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			resolved := getCode(l)(charpkt.AddCharacterEntryWriter, string(AddCharacterCodeOk), "codes", options)
			entry := toCharacterListEntry(c, false)
			return charpkt.NewAddCharacterEntry(resolved, entry).Encode(l, ctx)(options)
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

func AddCharacterErrorBody(code AddCharacterCode) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			resolved := getCode(l)(charpkt.AddCharacterEntryWriter, string(code), "codes", options)
			return charpkt.NewAddCharacterError(resolved).Encode(l, ctx)(options)
		}
	}
}
