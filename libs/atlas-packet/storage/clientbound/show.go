package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// Storage tab-flag bits (mirrors storage.StorageFlag). The Show packet body is
// segmented per set tab bit; meso is gated on the currency bit.
const (
	showFlagCurrency uint64 = 2
	showFlagEquip    uint64 = 4
	showFlagUse      uint64 = 8
	showFlagSetup    uint64 = 16
	showFlagEtc      uint64 = 32
	showFlagCash     uint64 = 64
)

// showTab pairs a tab bit with the inventory type whose assets it carries, in
// the client's read order (equip, use, setup, etc, cash).
type showTab struct {
	bit uint64
	typ inventory.Type
}

var showTabs = []showTab{
	{showFlagEquip, inventory.TypeValueEquip},
	{showFlagUse, inventory.TypeValueUse},
	{showFlagSetup, inventory.TypeValueSetup},
	{showFlagEtc, inventory.TypeValueETC},
	{showFlagCash, inventory.TypeValueCash},
}

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
		if m.flags&showFlagCurrency != 0 {
			w.WriteInt(m.meso)
		}
		for _, tab := range showTabs {
			if m.flags&tab.bit == 0 {
				continue
			}
			var bucket []model.Asset
			for _, a := range m.assets {
				if a.InventoryType() == tab.typ {
					bucket = append(bucket, a)
				}
			}
			w.WriteByte(byte(len(bucket)))
			for _, a := range bucket {
				w.WriteByteArray(a.Encode(l, ctx)(options))
			}
		}
		return w.Bytes()
	}
}

func (m *Show) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.npcId = r.ReadUint32()
		m.slots = r.ReadByte()
		m.flags = r.ReadUint64()
		if m.flags&showFlagCurrency != 0 {
			m.meso = r.ReadUint32()
		}
		m.assets = nil
		for _, tab := range showTabs {
			if m.flags&tab.bit == 0 {
				continue
			}
			count := int(r.ReadByte())
			for i := 0; i < count; i++ {
				var a model.Asset
				a.Decode(l, ctx)(r, options)
				m.assets = append(m.assets, a)
			}
		}
	}
}
