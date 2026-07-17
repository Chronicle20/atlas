package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// Discrete per-mode SetGetItems arms of the CTrunkDlg::OnPacket dispatcher
// (STORAGE op). STORE_ASSETS (gms mode 13 / jms 12) and RETRIEVE_ASSETS
// (gms mode 9 / jms 8) both route to the same CTrunkDlg::SetGetItems body —
// the dispatcher only differs by the leading mode byte — so the wire shape is
// identical: Decode1(mode), Decode1(slotCount), DecodeBuffer(8) tab-flag
// bitmask, Decode4 meso (gated on flag&2; runtime callers never set it), then
// a per-tab count byte + count*GW_ItemSlotBase::Decode loop over the set tab
// bits (4/8/16/32/64). Read order is IDA-confirmed identical across versions:
// SetGetItems v83 0x7c5dfd (dispatcher 0x7c8a4c), v84 sub via 0x7eec1a, v87
// via 0x81c336, v95 0x76a390 (dispatcher 0x76a990), jms via 0x84e5a1.
//
// task-096 discrete-per-mode rule: a packet that maps to one operation/mode
// gets its OWN struct that FIXES its own operation KEY (the body func in
// storage/operation_body.go resolves the per-tenant byte via WithResolvedCode
// for the FIXED key and passes it to the constructor — the caller never
// supplies a mode). The former shared UpdateAssets shape is retired. The
// version-shifted mode bytes trace to docs/packets/dispatchers/storage_operation.yaml.

// StoreAssets — the STORE_ASSETS SetGetItems arm (gms mode 13 / jms 12). Sent
// after a deposit so the client repaints the storage tab with its updated
// contents.
//
// packet-audit:fname CTrunkDlg::OnPacket#StoreAssets
type StoreAssets struct {
	mode   byte
	slots  byte
	flags  uint64
	assets []model.Asset
}

func NewStorageStoreAssets(mode byte, slots byte, flags uint64, assets []model.Asset) StoreAssets {
	return StoreAssets{mode: mode, slots: slots, flags: flags, assets: assets}
}

func (m StoreAssets) Mode() byte            { return m.mode }
func (m StoreAssets) Slots() byte           { return m.slots }
func (m StoreAssets) Flags() uint64         { return m.flags }
func (m StoreAssets) Assets() []model.Asset { return m.assets }
func (m StoreAssets) Operation() string     { return StorageOperationWriter }

func (m StoreAssets) String() string {
	return fmt.Sprintf("storage store assets slots [%d] entries [%d]", m.slots, len(m.assets))
}

func (m StoreAssets) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
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

func (m *StoreAssets) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
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

// RetrieveAssets — the RETRIEVE_ASSETS SetGetItems arm (gms mode 9 / jms 8).
// Sent after a withdrawal so the client repaints the storage tab with its
// updated contents.
//
// packet-audit:fname CTrunkDlg::OnPacket#RetrieveAssets
type RetrieveAssets struct {
	mode   byte
	slots  byte
	flags  uint64
	assets []model.Asset
}

func NewStorageRetrieveAssets(mode byte, slots byte, flags uint64, assets []model.Asset) RetrieveAssets {
	return RetrieveAssets{mode: mode, slots: slots, flags: flags, assets: assets}
}

func (m RetrieveAssets) Mode() byte            { return m.mode }
func (m RetrieveAssets) Slots() byte           { return m.slots }
func (m RetrieveAssets) Flags() uint64         { return m.flags }
func (m RetrieveAssets) Assets() []model.Asset { return m.assets }
func (m RetrieveAssets) Operation() string     { return StorageOperationWriter }

func (m RetrieveAssets) String() string {
	return fmt.Sprintf("storage retrieve assets slots [%d] entries [%d]", m.slots, len(m.assets))
}

func (m RetrieveAssets) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
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

func (m *RetrieveAssets) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
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
