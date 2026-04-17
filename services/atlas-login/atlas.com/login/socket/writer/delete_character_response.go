package writer

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/sirupsen/logrus"

	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
)


func DeleteCharacterResponseBody(characterId uint32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			resolved := getCode(l)(charpkt.DeleteCharacterResponseWriter, string(DeleteCharacterCodeOk), "codes", options)
			return charpkt.NewDeleteCharacterResponse(characterId, resolved).Encode(l, ctx)(options)
		}
	}
}

type DeleteCharacterCode string

const (
	DeleteCharacterCodeOk                             DeleteCharacterCode = "OK"
	DeleteCharacterCodeUnableToConnect                DeleteCharacterCode = "UNABLE_TO_CONNECT_SYSTEM_ERROR"
	DeleteCharacterCodeUnknownError                   DeleteCharacterCode = "UNKNOWN_ERROR"
	DeleteCharacterCodeTooManyConnections             DeleteCharacterCode = "TOO_MANY_CONNECTIONS"
	DeleteCharacterCodeNexonIdDifferent               DeleteCharacterCode = "NEXON_ID_DIFFERENT_THEN_REGISTERED"
	DeleteCharacterCodeCannotDeleteGuildMaster        DeleteCharacterCode = "CANNOT_DELETE_AS_GUILD_MASTER"
	DeleteCharacterCodeSecondaryPinMismatch           DeleteCharacterCode = "SECONDARY_PIN_DOES_NOT_MATCH"
	DeleteCharacterCodeCannotDeleteEngaged            DeleteCharacterCode = "CANNOT_DELETE_WHEN_ENGAGED"
	DeleteCharacterCodeOneTimePasswordMismatch        DeleteCharacterCode = "ONE_TIME_PASSWORD_DOES_NOT_MATCH"
	DeleteCharacterCodeOneTimePasswordAttemptExceeded DeleteCharacterCode = "ONE_TIME_PASSWORD_ATTEMPTS_EXCEEDED"
	DeleteCharacterCodeOneTimeServiceNotAvailable     DeleteCharacterCode = "ONE_TIME_PASSWORD_SERVICE_NOT_AVAILABLE"
	DeleteCharacterCodeOneTimeTrialEnded              DeleteCharacterCode = "ONE_TIME_PASSWORD_TRIAL_PERIOD_ENDED"
	DeleteCharacterCodeCannotDeleteInFamily           DeleteCharacterCode = "CANNOT_DELETE_WITH_FAMILY"
)

func DeleteCharacterErrorBody(characterId uint32, code DeleteCharacterCode) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			resolved := getCode(l)(charpkt.DeleteCharacterResponseWriter, string(code), "codes", options)
			return charpkt.NewDeleteCharacterResponse(characterId, resolved).Encode(l, ctx)(options)
		}
	}
}
