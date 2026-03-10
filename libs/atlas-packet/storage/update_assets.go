package storage

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// UpdateAssets - mode, slots, flags, assetEntryBytes
type UpdateAssets struct {
	mode            byte
	slots           byte
	flags           uint64
	assetEntryBytes [][]byte
}

func NewStorageUpdateAssets(mode byte, slots byte, flags uint64, assetEntryBytes [][]byte) UpdateAssets {
	return UpdateAssets{mode: mode, slots: slots, flags: flags, assetEntryBytes: assetEntryBytes}
}

func (m UpdateAssets) Mode() byte               { return m.mode }
func (m UpdateAssets) Slots() byte              { return m.slots }
func (m UpdateAssets) Flags() uint64            { return m.flags }
func (m UpdateAssets) AssetEntryBytes() [][]byte { return m.assetEntryBytes }
func (m UpdateAssets) Operation() string         { return StorageOperationWriter }

func (m UpdateAssets) String() string {
	return fmt.Sprintf("storage update assets slots [%d] entries [%d]", m.slots, len(m.assetEntryBytes))
}

func (m UpdateAssets) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.slots)
		w.WriteLong(m.flags)
		w.WriteByte(byte(len(m.assetEntryBytes)))
		for _, entry := range m.assetEntryBytes {
			w.WriteByteArray(entry)
		}
		return w.Bytes()
	}
}

func (m *UpdateAssets) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		// No-op: server-send-only
	}
}
