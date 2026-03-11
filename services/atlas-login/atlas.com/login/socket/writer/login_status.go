package writer

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"

	loginpkt "github.com/Chronicle20/atlas-packet/login"
)


const (
	Banned                     = "BANNED"
	DeletedOrBlocked           = "DELETED_OR_BLOCKED"
	IncorrectPassword          = "INCORRECT_PASSWORD"
	NotRegistered              = "NOT_REGISTERED"
	SystemError1               = "SYSTEM_ERROR_1"
	AlreadyLoggedIn            = "ALREADY_LOGGED_IN"
	SystemError2               = "SYSTEM_ERROR_2"
	SystemError3               = "SYSTEM_ERROR_3"
	TooManyConnections         = "TOO_MANY_CONNECTIONS"
	AgeLimit                   = "AGE_LIMIT"
	UnableToLogOnAsMasterIp    = "UNABLE_TO_LOG_ON_AS_MASTER_AT_IP"
	WrongGateway               = "WRONG_GATEWAY"
	ProcessingRequest          = "PROCESSING_REQUEST"
	AccountVerificationNeeded  = "ACCOUNT_VERIFICATION_NEEDED"
	WrongPersonalInformation   = "WRONG_PERSONAL_INFORMATION"
	AccountVerificationNeeded2 = "ACCOUNT_VERIFICATION_NEEDED_2"
	LicenseAgreement           = "LICENSE_AGREEMENT"
	MapleEuropeNotice          = "MAPLE_EUROPE_NOTICE"
	FullClientNotice           = "FULL_CLIENT_NOTICE"
)

func AuthSuccessBody(accountId uint32, name string, gender byte, usesPin bool, pic string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			return loginpkt.NewAuthSuccess(accountId, name, gender, usesPin, pic).Encode(l, ctx)(options)
		}
	}
}

func AuthTemporaryBanBody(until time.Time, reason byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			resolved := getCode(l)(loginpkt.AuthLoginFailedWriter, Banned, "failedReasonCodes", options)
			return loginpkt.NewAuthTemporaryBan(resolved, reason, until).Encode(l, ctx)(options)
		}
	}
}

func AuthPermanentBanBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			resolved := getCode(l)(loginpkt.AuthLoginFailedWriter, Banned, "failedReasonCodes", options)
			return loginpkt.NewAuthPermanentBan(resolved).Encode(l, ctx)(options)
		}
	}
}

func AuthLoginFailedBody(reason string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			resolved := getCode(l)(loginpkt.AuthLoginFailedWriter, reason, "failedReasonCodes", options)
			return loginpkt.NewAuthLoginFailed(resolved).Encode(l, ctx)(options)
		}
	}
}
