package writer

import (
	"atlas-login/character"
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const AddCharacterEntry = "AddCharacterEntry"

func AddCharacterEntryBody(c character.Model) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCode(l)(AddCharacterEntry, string(AddCharacterCodeOk), "codes", options))
			WriteCharacter(l, ctx)(w, options)(c, false)
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

func AddCharacterErrorBody(code AddCharacterCode) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(getCode(l)(AddCharacterEntry, string(code), "codes", options))
			return w.Bytes()
		}
	}
}
