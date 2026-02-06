package handler

import (
	as "atlas-login/account/session"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const LoginHandle = "LoginHandle"

type LoginRequest struct {
	name           string
	password       string
	hwid           []byte
	gameRoomClient uint32
	gameStartMode  byte
	unknown1       byte
}

func (l *LoginRequest) Name() string {
	return l.name
}

func (l *LoginRequest) Password() string {
	return l.password
}

func (l *LoginRequest) GameRoomClient() uint32 {
	return l.gameRoomClient
}

func (l *LoginRequest) GameStartMode() byte {
	return l.gameStartMode
}

func ReadLoginRequest(reader *request.Reader) *LoginRequest {
	name := reader.ReadAsciiString()
	password := reader.ReadAsciiString()
	hwid := reader.ReadBytes(16)
	gameRoomClient := reader.ReadUint32()
	gameStartMode := reader.ReadByte()
	unknown1 := reader.ReadByte()

	return &LoginRequest{
		name:           name,
		password:       password,
		hwid:           hwid,
		gameRoomClient: gameRoomClient,
		gameStartMode:  gameStartMode,
		unknown1:       unknown1,
	}
}

func LoginHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader) {
	t := tenant.MustFromContext(ctx)
	return func(s session.Model, r *request.Reader) {
		p := ReadLoginRequest(r)
		l.Debugf("Reading [%s] message. body={name=%s, gameRoomClient=%d, gameStartMode=%d}", LoginHandle, p.Name(), p.GameRoomClient(), p.GameStartMode())

		err := as.NewProcessor(l, ctx).Create(s.SessionId(), s.AccountId(), p.Name(), p.Password(), "")
		if err != nil {
			authLoginFailedFunc := session.Announce(l)(wp)(writer.AuthLoginFailed)
			err = authLoginFailedFunc(s, writer.AuthLoginFailedBody(l, t)(writer.SystemError1))
			if err != nil {
				l.WithError(err).Errorf("Unable to issue [%s].", writer.AuthLoginFailed)
			}
			return
		}
	}
}
