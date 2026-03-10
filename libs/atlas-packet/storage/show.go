package storage

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// Show - mode, npcId, slots, flags, meso, assetEntryBytes
type Show struct {
	mode            byte
	npcId           uint32
	slots           byte
	flags           uint64
	meso            uint32
	assetEntryBytes [][]byte
}

func NewStorageShow(mode byte, npcId uint32, slots byte, flags uint64, meso uint32, assetEntryBytes [][]byte) Show {
	return Show{mode: mode, npcId: npcId, slots: slots, flags: flags, meso: meso, assetEntryBytes: assetEntryBytes}
}

func (m Show) Mode() byte               { return m.mode }
func (m Show) NpcId() uint32             { return m.npcId }
func (m Show) Slots() byte              { return m.slots }
func (m Show) Flags() uint64            { return m.flags }
func (m Show) Meso() uint32             { return m.meso }
func (m Show) AssetEntryBytes() [][]byte { return m.assetEntryBytes }
func (m Show) Operation() string         { return StorageOperationWriter }

func (m Show) String() string {
	return fmt.Sprintf("storage show npcId [%d] slots [%d] assets [%d]", m.npcId, m.slots, len(m.assetEntryBytes))
}

func (m Show) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.npcId)
		w.WriteByte(m.slots)
		w.WriteLong(m.flags)
		w.WriteInt(m.meso)
		w.WriteShort(0)
		w.WriteByte(byte(len(m.assetEntryBytes)))
		for _, entry := range m.assetEntryBytes {
			w.WriteByteArray(entry)
		}
		w.WriteShort(0)
		w.WriteByte(0)
		return w.Bytes()
	}
}

func (m *Show) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		// No-op: server-send-only
	}
}
