package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const AfterLoginHandle = "AfterLoginHandle"

// AfterLogin - CLogin::OnSetAccountResult - CLogin::OnCheckPinCodeResult - CLogin::OnCheckPasswordResult - CLogin::OnSelectWorldResult
type AfterLogin struct {
	pinMode   byte
	opt2      byte // 0 in OnCheckPinCodeResult
	accountId uint32
	pin       string
}

func (m AfterLogin) PinMode() byte {
	return m.pinMode
}

func (m AfterLogin) Opt2() byte {
	return m.opt2
}

func (m AfterLogin) AccountId() uint32 {
	return m.accountId
}

func (m AfterLogin) Pin() string {
	return m.pin
}

func (m AfterLogin) Operation() string {
	return AfterLoginHandle
}

func (m AfterLogin) String() string {
	return fmt.Sprintf("pinMode [%d], opt2 [%d], pin [%s]", m.pinMode, m.opt2, m.pin)
}

func (m AfterLogin) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.PinMode())
		if m.PinMode() > 0 {
			w.WriteByte(m.Opt2())
			// Legacy GMS (< v83) AFTER_LOGIN carries the accountId int between opt2
			// and the pin string. v79 send sites CLogin::OnSetAccountResult @0x5d0800
			// and CLogin::OnCheckPinCodeResult @0x5d0aaf/@0x5d09be build COutPacket(9)
			// as Encode1(pinMode)+Encode1(opt2)+Encode4(accountId @g_pWvsContext+8232)+
			// EncodeStr(pin). v83 @0x5fc731 and v84/87/95 omit the int, so gate it to
			// the legacy range only.
			if t.Region() == "GMS" && t.MajorVersion() < 83 {
				w.WriteInt(m.AccountId())
			}
			w.WriteAsciiString(m.Pin())
		}
		return w.Bytes()
	}
}

func (m *AfterLogin) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		t := tenant.MustFromContext(ctx)
		m.pinMode = r.ReadByte()
		if m.pinMode > 0 {
			m.opt2 = r.ReadByte()
			if t.Region() == "GMS" && t.MajorVersion() < 83 {
				m.accountId = r.ReadUint32()
			}
			m.pin = r.ReadAsciiString()
		}
	}
}
