package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const NpcImitatedDataWriter = "ImitatedNPCData"

// packet-audit:fname CNpcPool::OnNpcImitateData
type ImitatedNpc struct {
	templateId uint32
	name       string
	look       packetmodel.Avatar
}

func NewImitatedNpc(templateId uint32, name string, look packetmodel.Avatar) ImitatedNpc {
	return ImitatedNpc{templateId: templateId, name: name, look: look}
}

func (m ImitatedNpc) TemplateId() uint32       { return m.templateId }
func (m ImitatedNpc) Name() string             { return m.name }
func (m ImitatedNpc) Look() packetmodel.Avatar { return m.look }

type ImitatedNpcData struct {
	entries []ImitatedNpc
}

func NewImitatedNpcData(entries []ImitatedNpc) ImitatedNpcData {
	return ImitatedNpcData{entries: entries}
}

func (m ImitatedNpcData) Operation() string { return NpcImitatedDataWriter }
func (m ImitatedNpcData) String() string {
	return fmt.Sprintf("entries [%d]", len(m.entries))
}

func (m ImitatedNpcData) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(len(m.entries)))
		for _, e := range m.entries {
			w.WriteInt(e.templateId)
			w.WriteAsciiString(e.name)
			w.WriteByteArray(e.look.Encode(l, ctx)(options))
		}
		return w.Bytes()
	}
}

func (m *ImitatedNpcData) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		count := r.ReadByte()
		entries := make([]ImitatedNpc, 0, count)
		for i := byte(0); i < count; i++ {
			templateId := r.ReadUint32()
			name := r.ReadAsciiString()
			look := packetmodel.Avatar{}
			look.Decode(l, ctx)(r, options)
			entries = append(entries, NewImitatedNpc(templateId, name, look))
		}
		m.entries = entries
	}
}
