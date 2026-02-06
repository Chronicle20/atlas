package session

import (
	"strconv"

	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

func WriteHello(l logrus.FieldLogger) func(majorVersion uint16, minorVersion uint16, sendIv []byte, recvIv []byte, locale byte) []byte {
	return func(majorVersion uint16, minorVersion uint16, sendIv []byte, recvIv []byte, locale byte) []byte {
		w := response.NewWriter(l)
		w.WriteShort(uint16(0x0E))
		w.WriteShort(majorVersion)
		w.WriteAsciiString(strconv.Itoa(int(minorVersion)))
		w.WriteByteArray(recvIv)
		w.WriteByteArray(sendIv)
		w.WriteByte(locale)
		return w.Bytes()
	}
}
