package handler

import (
	"atlas-login/account"
	"atlas-login/session"
	"atlas-login/socket/writer"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

const RegisterPicHandle = "RegisterPicHandle"

func RegisterPicHandleFunc(l logrus.FieldLogger, span opentracing.Span, wp writer.Producer) func(s session.Model, r *request.Reader) {
	return func(s session.Model, r *request.Reader) {
		opt := r.ReadByte()
		characterId := r.ReadUint32()
		sMacAddressWithHDDSerial := r.ReadAsciiString()
		sMacAddressWithHDDSerial2 := r.ReadAsciiString()
		pic := r.ReadAsciiString()

		l.Debugf("Attempting to register PIC [%s]. opt [%d], character [%d], hwid [%s] hwid [%s].", pic, opt, characterId, sMacAddressWithHDDSerial, sMacAddressWithHDDSerial2)

		a, err := account.GetById(l, span, s.Tenant())(s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Failed to get account by id [%d].", s.AccountId())
			//TODO
			return
		}
		if a.PIC() != "" {
			l.Warnf("Account [%d] already has PIC.", s.AccountId())
			//TODO
			return
		}
		err = account.UpdatePic(l, span, s.Tenant())(s.AccountId(), pic)
		if err != nil {
			l.WithError(err).Errorf("Unable to register PIC [%s] for account [%d].", pic, s.AccountId())
		}
		// TODO announce server ip.
	}
}
