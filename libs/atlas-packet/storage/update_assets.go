package storage

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// UpdateAssets - mode, slots, flags, assets
type UpdateAssets struct {
	mode   byte
	slots  byte
	flags  uint64
	assets []model.Asset
}

func NewStorageUpdateAssets(mode byte, slots byte, flags uint64, assets []model.Asset) UpdateAssets {
	return UpdateAssets{mode: mode, slots: slots, flags: flags, assets: assets}
}

func (m UpdateAssets) Mode() byte           { return m.mode }
func (m UpdateAssets) Slots() byte          { return m.slots }
func (m UpdateAssets) Flags() uint64        { return m.flags }
func (m UpdateAssets) Assets() []model.Asset { return m.assets }
func (m UpdateAssets) Operation() string     { return StorageOperationWriter }

func (m UpdateAssets) String() string {
	return fmt.Sprintf("storage update assets slots [%d] entries [%d]", m.slots, len(m.assets))
}

func (m UpdateAssets) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.slots)
		w.WriteLong(m.flags)
		w.WriteByte(byte(len(m.assets)))
		for _, a := range m.assets {
			w.WriteByteArray(a.Encode(l, ctx)(options))
		}
		return w.Bytes()
	}
}

func (m *UpdateAssets) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.slots = r.ReadByte()
		m.flags = r.ReadUint64()
		count := int(r.ReadByte())
		m.assets = make([]model.Asset, count)
		for i := 0; i < count; i++ {
			m.assets[i].Decode(l, ctx)(r, options)
		}
	}
}
