package writer

import (
	"atlas-login/tenant"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const AuthSuccess = "AuthSuccess"
const AuthTemporaryBan = "AuthTemporaryBan"
const AuthPermanentBan = "AuthPermanentBan"
const AuthLoginFailed = "AuthLoginFailed"

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

func AuthSuccessBody(l logrus.FieldLogger, tenant tenant.Model) func(accountId uint32, name string, gender byte, pin string, pic string) BodyProducer {
	return func(accountId uint32, name string, gender byte, pin string, pic string) BodyProducer {
		return func(op uint16, _ map[string]interface{}) []byte {
			w := response.NewWriter(l)
			w.WriteShort(op)
			w.WriteByte(0) // success
			w.WriteByte(0)

			if tenant.Region == "GMS" {
				w.WriteInt(0)
			}

			w.WriteInt(accountId)
			w.WriteByte(gender)

			//boolean canFly = false;// Server.getInstance().canFly(client.getAccID());
			//writer.writeBool((YamlConfig.config.server.USE_ENFORCE_ADMIN_ACCOUNT || canFly) && client.getGMLevel() > 1);    // GM
			w.WriteBool(false)

			//writer.write(((YamlConfig.config.server.USE_ENFORCE_ADMIN_ACCOUNT || canFly) && client.getGMLevel() > 1) ? 0x80 : 0);  //
			// Admin Byte. 0x80,0x40,0x20.. Rubbish.
			w.WriteByte(0)

			if tenant.Region == "GMS" {
				// country code
				w.WriteByte(0)
				w.WriteAsciiString(name)
				w.WriteByte(0)
				// quiet ban
				w.WriteByte(0)
				// quiet ban timestamp
				w.WriteLong(0)
				// creation timestamp
				w.WriteLong(0)
				// nNumOfCharacter
				w.WriteInt(1)
				// 0 = Pin-System Enabled, 1 = Disabled
				w.WriteByte(1)
				// 0 = Register PIC, 1 = Ask for PIC, 2 = Disabled
				w.WriteByte(2)

				if tenant.MajorVersion >= 87 {
					w.WriteLong(0)
				}
			} else if tenant.Region == "JMS" {
				w.WriteAsciiString(name)
				w.WriteAsciiString(name)
				w.WriteByte(0)
				w.WriteByte(0)
				w.WriteByte(0)
				w.WriteByte(0)
				w.WriteByte(0)
				w.WriteByte(0)
				w.WriteLong(0)
				w.WriteAsciiString(name)
			}

			return w.Bytes()
		}
	}
}

func AuthTemporaryBanBody(l logrus.FieldLogger, tenant tenant.Model) func(until uint64, reason byte) BodyProducer {
	return func(until uint64, reason byte) BodyProducer {
		return func(op uint16, options map[string]interface{}) []byte {
			w := response.NewWriter(l)
			w.WriteShort(op)
			code := getFailedReason(l)(Banned, options)
			w.WriteByte(code)
			w.WriteByte(0)

			if tenant.Region == "GMS" {
				w.WriteInt(0)
			}

			w.WriteByte(reason)
			w.WriteLong(until) // Temp ban date is handled as a 64-bit long, number of 100NS intervals since 1/1/1601.

			rtn := w.Bytes()
			l.Debugf("Writing [%s] message. opcode [0x%02X]. body={reason=%d, until=%d}.", AuthTemporaryBan, op&0xFF, reason, until)
			return rtn
		}
	}
}

func AuthPermanentBanBody(l logrus.FieldLogger, tenant tenant.Model) BodyProducer {
	return func(op uint16, options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteShort(op)
		code := getFailedReason(l)(Banned, options)
		w.WriteByte(code)
		w.WriteByte(0)

		if tenant.Region == "GMS" {
			w.WriteInt(0)
		}

		w.WriteByte(0)
		w.WriteLong(0)

		rtn := w.Bytes()
		l.Debugf("Writing [%s] message. opcode [0x%02X].", AuthPermanentBan, op&0xFF)
		return rtn
	}
}

func AuthLoginFailedBody(l logrus.FieldLogger, tenant tenant.Model) func(reason string) BodyProducer {
	return func(reason string) BodyProducer {
		return func(op uint16, options map[string]interface{}) []byte {
			w := response.NewWriter(l)
			w.WriteShort(op)
			code := getFailedReason(l)(reason, options)
			w.WriteByte(code)
			w.WriteByte(0)

			if tenant.Region == "GMS" {
				w.WriteInt(0)
			}

			rtn := w.Bytes()
			l.Debugf("Writing [%s] message. opcode [0x%02X]. reason=[%s]. body={reason=%d}.", AuthLoginFailed, op&0xFF, reason, code)
			return rtn
		}
	}
}

const failedReasonCodeProperty = "failedReasonCodes"

func getFailedReason(l logrus.FieldLogger) func(reason string, options map[string]interface{}) byte {
	return func(reason string, options map[string]interface{}) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options[failedReasonCodeProperty]; !ok {
			l.Errorf("Reason code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", reason, AuthLoginFailed)
			return 99
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Reason code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", reason, AuthLoginFailed)
			return 99
		}

		code, ok := codes[reason].(float64)
		if !ok {
			l.Errorf("Reason code [%s] not configured for use in [%s]. Defaulting to 99 which will likely cause a client crash.", reason, AuthLoginFailed)
			return 99
		}
		return byte(code)
	}
}
