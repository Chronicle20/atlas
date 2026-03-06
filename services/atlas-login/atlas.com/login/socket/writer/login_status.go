package writer

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
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

func AuthSuccessBody(accountId uint32, name string, gender byte, usesPin bool, pic string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			w.WriteByte(0) // success
			w.WriteByte(0)

			if t.Region() == "GMS" {
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

			if t.Region() == "GMS" {
				// country code
				if t.MajorVersion() > 12 {
					w.WriteByte(0)
				}
				w.WriteAsciiString(name)

				if t.MajorVersion() > 12 {
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
					w.WriteBool(!usesPin)
					// 0 = Register PIC, 1 = Ask for PIC, 2 = Disabled (disables character deletion without client edit).
					var needsPic = byte(0)
					if pic != "" {
						needsPic = byte(1)
					}
					w.WriteByte(needsPic)
				} else {
					w.WriteLong(0)
					w.WriteLong(0)
					w.WriteLong(0)
				}

				if t.MajorVersion() >= 87 {
					w.WriteLong(0)
				}
			} else if t.Region() == "JMS" {
				w.WriteAsciiString(name)
				w.WriteAsciiString(name)
				w.WriteByte(0)
				w.WriteByte(0)
				w.WriteByte(0)
				w.WriteByte(0)
				w.WriteByte(0) // enables secure password
				w.WriteByte(0)
				w.WriteLong(0)
				w.WriteAsciiString(name)
			}
			return w.Bytes()
		}
	}
}

func AuthTemporaryBanBody(until time.Time, reason byte) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			code := getCode(l)(AuthLoginFailed, Banned, "failedReasonCodes", options)
			w.WriteByte(code)
			w.WriteByte(0)

			if t.Region() == "GMS" {
				w.WriteInt(0)
			}

			w.WriteByte(reason)
			w.WriteLong(uint64(msTime(until)))
			return w.Bytes()
		}
	}
}

func msTime(t time.Time) int64 {
	if t.IsZero() {
		return -1
	}
	return t.Unix()*int64(10000000) + int64(116444736000000000)
}

func AuthPermanentBanBody() packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			code := getCode(l)(AuthLoginFailed, Banned, "failedReasonCodes", options)
			w.WriteByte(code)
			w.WriteByte(0)

			if t.Region() == "GMS" {
				w.WriteInt(0)
			}

			w.WriteByte(0)
			w.WriteLong(0)
			return w.Bytes()
		}
	}
}

func AuthLoginFailedBody(reason string) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			code := getCode(l)(AuthLoginFailed, reason, "failedReasonCodes", options)
			w.WriteByte(code)
			w.WriteByte(0)

			if t.Region() == "GMS" {
				w.WriteInt(0)
			}
			return w.Bytes()
		}
	}
}
