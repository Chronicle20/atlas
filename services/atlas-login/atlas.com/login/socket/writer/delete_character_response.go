package writer

import (
	"atlas-login/tenant"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const DeleteCharacterResponse = "DeleteCharacterResponse"

func DeleteCharacterResponseBody(l logrus.FieldLogger, _ tenant.Model) func(characterId uint32) BodyProducer {
	return func(characterId uint32) BodyProducer {
		return func(op uint16, options map[string]interface{}) []byte {
			w := response.NewWriter(l)
			w.WriteShort(op)
			w.WriteInt(characterId)
			w.WriteByte(getDeleteCharacterCode(l)(DeleteCharacterCodeOk, options))
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
		return func(op uint16, options map[string]interface{}) []byte {
			w := response.NewWriter(l)
			w.WriteShort(op)
			w.WriteInt(characterId)
			w.WriteByte(getDeleteCharacterCode(l)(code, options))
			return w.Bytes()
		}
	}
}

const DeleteCharacterCodeProperty = "codes"

func getDeleteCharacterCode(l logrus.FieldLogger) func(code DeleteCharacterCode, options map[string]interface{}) byte {
	return func(codeString DeleteCharacterCode, options map[string]interface{}) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options[DeleteCharacterCodeProperty]; !ok {
			l.Errorf("Reason code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", codeString, DeleteCharacterResponse)
			return 99
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Reason code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", codeString, DeleteCharacterResponse)
			return 99
		}

		code, ok := codes[string(codeString)].(float64)
		if !ok {
			l.Errorf("Reason code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", codeString, DeleteCharacterResponse)
			return 99
		}
		return byte(code)
	}
}
