package login

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const ServerIPWriter = "ServerIP"

type ServerIP struct {
	code     byte
	mode     byte
	ipAddr   string
	port     uint16
	clientId uint32
}

func NewServerIP(code byte, mode byte, ipAddr string, port uint16, clientId uint32) ServerIP {
	return ServerIP{code: code, mode: mode, ipAddr: ipAddr, port: port, clientId: clientId}
}

func NewServerIPError(code byte, mode byte) ServerIP {
	return ServerIP{code: code, mode: mode}
}

func (m ServerIP) Code() byte        { return m.code }
func (m ServerIP) Mode() byte        { return m.mode }
func (m ServerIP) IpAddr() string    { return m.ipAddr }
func (m ServerIP) Port() uint16      { return m.port }
func (m ServerIP) ClientId() uint32  { return m.clientId }
func (m ServerIP) Operation() string { return ServerIPWriter }
func (m ServerIP) String() string {
	return fmt.Sprintf("code [%d], mode [%d], ipAddr [%s], port [%d], clientId [%d]", m.code, m.mode, m.ipAddr, m.port, m.clientId)
}

func ipAsByteArray(ipAddress string) []byte {
	var ob = make([]byte, 0)
	os := strings.Split(ipAddress, ".")
	for _, x := range os {
		o, err := strconv.ParseUint(x, 10, 8)
		if err == nil {
			ob = append(ob, byte(o))
		}
	}
	return ob
}

func ipFromByteArray(b []byte) string {
	parts := make([]string, len(b))
	for i, v := range b {
		parts[i] = strconv.Itoa(int(v))
	}
	return strings.Join(parts, ".")
}

func (m ServerIP) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.code)
		w.WriteByte(m.mode)
		if m.ipAddr != "" {
			w.WriteByteArray(ipAsByteArray(m.ipAddr))
			w.WriteShort(m.port)
			w.WriteInt(m.clientId)
			w.WriteByte(0) // bAuthenCode
			if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
				w.WriteInt(0) // ulPremiumArgument
			}
		}
		return w.Bytes()
	}
}

func (m *ServerIP) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.code = r.ReadByte()
		m.mode = r.ReadByte()
		if r.Available() > 0 {
			m.ipAddr = ipFromByteArray(r.ReadBytes(4))
			m.port = r.ReadUint16()
			m.clientId = r.ReadUint32()
			_ = r.ReadByte() // bAuthenCode
			if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
				_ = r.ReadUint32() // ulPremiumArgument
			}
		}
	}
}
