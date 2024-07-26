package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	"strconv"
	"strings"
)

const ChannelChange = "ChannelChange"

func ChannelChangeBody(ipAddr string, port uint16) BodyProducer {
	return func(w *response.Writer, options map[string]interface{}) []byte {
		w.WriteByte(1)
		ob := ipAsByteArray(ipAddr)
		w.WriteByteArray(ob)
		w.WriteShort(port)
		return w.Bytes()
	}
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
