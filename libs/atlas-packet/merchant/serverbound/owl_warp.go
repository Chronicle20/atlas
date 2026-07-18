package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const OwlWarpHandle = "OwlWarpHandle"

// packet-audit:fname CUIShopScanResult::OnButtonClicked
// OwlWarp is sent when the player clicks a shop-scanner result row. The two
// ints echo the record's dwMiniRoomSN (Atlas sends the shop-owner characterId
// there) and dwFieldID verbatim (v83 sub_8A4423, v95 0x848e80).
type OwlWarp struct {
	ownerId uint32
	mapId   uint32
}

func NewOwlWarp(ownerId uint32, mapId uint32) OwlWarp {
	return OwlWarp{ownerId: ownerId, mapId: mapId}
}

func (m OwlWarp) OwnerId() uint32 {
	return m.ownerId
}

func (m OwlWarp) MapId() uint32 {
	return m.mapId
}

func (m OwlWarp) Operation() string {
	return OwlWarpHandle
}

func (m OwlWarp) String() string {
	return fmt.Sprintf("ownerId [%d] mapId [%d]", m.ownerId, m.mapId)
}

func (m OwlWarp) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerId)
		w.WriteInt(m.mapId)
		return w.Bytes()
	}
}

func (m *OwlWarp) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerId = r.ReadUint32()
		m.mapId = r.ReadUint32()
	}
}
