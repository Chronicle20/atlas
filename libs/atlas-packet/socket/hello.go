package socket

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const HelloWriter = "Hello"

// Hello - Initial handshake packet sent to client on connect.
type Hello struct {
	majorVersion uint16
	minorVersion uint16
	sendIv       []byte
	recvIv       []byte
	locale       byte
}

func NewHello(majorVersion uint16, minorVersion uint16, sendIv []byte, recvIv []byte, locale byte) Hello {
	return Hello{
		majorVersion: majorVersion,
		minorVersion: minorVersion,
		sendIv:       sendIv,
		recvIv:       recvIv,
		locale:       locale,
	}
}

func (m Hello) MajorVersion() uint16 { return m.majorVersion }
func (m Hello) MinorVersion() uint16 { return m.minorVersion }
func (m Hello) SendIv() []byte       { return m.sendIv }
func (m Hello) RecvIv() []byte       { return m.recvIv }
func (m Hello) Locale() byte         { return m.locale }

func (m Hello) Operation() string {
	return HelloWriter
}

func (m Hello) String() string {
	return fmt.Sprintf("majorVersion [%d], minorVersion [%d], locale [%d]", m.majorVersion, m.minorVersion, m.locale)
}

func (m Hello) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteShort(uint16(0x0E))
		w.WriteShort(m.majorVersion)
		w.WriteAsciiString(strconv.Itoa(int(m.minorVersion)))
		w.WriteByteArray(m.recvIv)
		w.WriteByteArray(m.sendIv)
		w.WriteByte(m.locale)
		return w.Bytes()
	}
}

func (m *Hello) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		_ = r.ReadUint16() // 0x0E
		m.majorVersion = r.ReadUint16()
		minorStr := r.ReadAsciiString()
		minor, _ := strconv.Atoi(minorStr)
		m.minorVersion = uint16(minor)
		m.recvIv = r.ReadBytes(4)
		m.sendIv = r.ReadBytes(4)
		m.locale = r.ReadByte()
	}
}
