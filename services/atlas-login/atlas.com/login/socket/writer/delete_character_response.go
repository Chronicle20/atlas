package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const DeleteCharacterResponse = "DeleteCharacterResponse"

func DeleteCharacterResponseBody(l logrus.FieldLogger, _ tenant.Model) func(characterId uint32) BodyProducer {
	return func(characterId uint32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			w.WriteByte(getCode(l)(DeleteCharacterResponse, string(DeleteCharacterCodeOk), "codes", options))
			return w.Bytes()
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

func DeleteCharacterErrorBody(l logrus.FieldLogger, _ tenant.Model) func(characterId uint32, code DeleteCharacterCode) BodyProducer {
	return func(characterId uint32, code DeleteCharacterCode) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteInt(characterId)
			w.WriteByte(getCode(l)(DeleteCharacterResponse, string(code), "codes", options))
			return w.Bytes()
		}
	}
}
