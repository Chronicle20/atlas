package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// ItemUseStoreSearch is the arm tail for USE_CASH_ITEM itemType 523 (Owl of
// Minerva, cash-slot type 29). CUIShopScanner::SendScanPacket appends
// [int searchItemId][byte bDescendingOrder][int updateTime] to the stashed
// use packet unconditionally in both v83 (sub_8A2407) and v95 (0x83f6b0);
// the GMS>=95 leading-updateTime gate lives in the ItemUse prefix codec.
type ItemUseStoreSearch struct {
	searchItemId uint32
	descending   bool
	updateTime   uint32
}

func NewItemUseStoreSearch(searchItemId uint32, descending bool, updateTime uint32) *ItemUseStoreSearch {
	return &ItemUseStoreSearch{searchItemId: searchItemId, descending: descending, updateTime: updateTime}
}

func (m ItemUseStoreSearch) SearchItemId() uint32 {
	return m.searchItemId
}

func (m ItemUseStoreSearch) Descending() bool {
	return m.descending
}

func (m ItemUseStoreSearch) UpdateTime() uint32 {
	return m.updateTime
}

func (m ItemUseStoreSearch) Operation() string {
	return "ItemUseStoreSearch"
}

func (m ItemUseStoreSearch) String() string {
	return fmt.Sprintf("searchItemId [%d] descending [%t] updateTime [%d]", m.searchItemId, m.descending, m.updateTime)
}

func (m ItemUseStoreSearch) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.searchItemId)
		w.WriteBool(m.descending)
		w.WriteInt(m.updateTime)
		return w.Bytes()
	}
}

func (m *ItemUseStoreSearch) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.searchItemId = r.ReadUint32()
		m.descending = r.ReadBool()
		m.updateTime = r.ReadUint32()
	}
}
