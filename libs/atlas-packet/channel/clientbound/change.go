package clientbound

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ChannelChangeWriter = "ChannelChange"

type ChannelChange struct {
	ipAddr string
	port   uint16
}

func NewChannelChange(ipAddr string, port uint16) ChannelChange {
	return ChannelChange{ipAddr: ipAddr, port: port}
}

func (m ChannelChange) IpAddr() string    { return m.ipAddr }
func (m ChannelChange) Port() uint16      { return m.port }
func (m ChannelChange) Operation() string { return ChannelChangeWriter }
func (m ChannelChange) String() string {
	return fmt.Sprintf("ipAddr [%s], port [%d]", m.ipAddr, m.port)
}

func channelIpAsByteArray(ipAddress string) []byte {
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

func channelIpFromByteArray(b []byte) string {
	parts := make([]string, len(b))
	for i, v := range b {
		parts[i] = strconv.Itoa(int(v))
	}
	return strings.Join(parts, ".")
}

func (m ChannelChange) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(1)
		w.WriteByteArray(channelIpAsByteArray(m.ipAddr))
		w.WriteShort(m.port)
		return w.Bytes()
	}
}

func (m *ChannelChange) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		_ = r.ReadByte() // 1
		m.ipAddr = channelIpFromByteArray(r.ReadBytes(4))
		m.port = r.ReadUint16()
	}
}
