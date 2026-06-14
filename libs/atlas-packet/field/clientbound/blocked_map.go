package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const BlockedMapWriter = "BlockedMap"

type BlockedMap struct {
	reason byte
}

func NewBlockedMap(reason byte) BlockedMap {
	return BlockedMap{reason: reason}
}

func (m BlockedMap) Reason() byte { return m.reason }

func (m BlockedMap) Operation() string { return BlockedMapWriter }
func (m BlockedMap) String() string {
	return fmt.Sprintf("reason [%d]", m.reason)
}

func (m BlockedMap) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.reason)
		return w.Bytes()
	}
}

func (m *BlockedMap) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.reason = r.ReadByte()
	}
}
