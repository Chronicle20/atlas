package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// Show - mode, npcId, slots, flags, meso, assets
type Show struct {
	mode   byte
	npcId  uint32
	slots  byte
	flags  uint64
	meso   uint32
	assets []model.Asset
}

func NewStorageShow(mode byte, npcId uint32, slots byte, flags uint64, meso uint32, assets []model.Asset) Show {
	return Show{mode: mode, npcId: npcId, slots: slots, flags: flags, meso: meso, assets: assets}
}

func (m Show) Mode() byte           { return m.mode }
func (m Show) NpcId() uint32        { return m.npcId }
func (m Show) Slots() byte          { return m.slots }
func (m Show) Flags() uint64        { return m.flags }
func (m Show) Meso() uint32         { return m.meso }
func (m Show) Assets() []model.Asset { return m.assets }
func (m Show) Operation() string     { return StorageOperationWriter }

func (m Show) String() string {
	return fmt.Sprintf("storage show npcId [%d] slots [%d] assets [%d]", m.npcId, m.slots, len(m.assets))
}

func (m Show) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.npcId)
		w.WriteByte(m.slots)
		w.WriteLong(m.flags)
		w.WriteInt(m.meso)
		w.WriteShort(0)
		w.WriteByte(byte(len(m.assets)))
		for _, a := range m.assets {
			w.WriteByteArray(a.Encode(l, ctx)(options))
		}
		w.WriteShort(0)
		w.WriteByte(0)
		return w.Bytes()
	}
}

func (m *Show) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.npcId = r.ReadUint32()
		m.slots = r.ReadByte()
		m.flags = r.ReadUint64()
		m.meso = r.ReadUint32()
		_ = r.ReadUint16() // padding
		count := int(r.ReadByte())
		m.assets = make([]model.Asset, count)
		for i := 0; i < count; i++ {
			m.assets[i].Decode(l, ctx)(r, options)
		}
		_ = r.ReadUint16() // padding
		_ = r.ReadByte()   // padding
	}
}
