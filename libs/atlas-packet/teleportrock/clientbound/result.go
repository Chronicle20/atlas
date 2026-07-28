package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const MapTransferResultWriter = "MapTransferResult"

// MapTransferList is the list-refresh form of MAP_TRANSFER_RESULT (modes 2/3):
// byte mode, byte targetList (0=regular 1=VIP), then exactly 5 (regular) or 10
// (VIP) x int mapId padded with EmptyMapId. The client reloads
// adwMapTransfer[5] / adwMapTransferEx[10] from this packet (design §1 Q4;
// identical v83 0xA25268 / v95 0x9F9F90).
type MapTransferList struct {
	mode byte
	vip  bool
	maps []_map.Id
}

func NewMapTransferList(mode byte, vip bool, maps []_map.Id) MapTransferList {
	return MapTransferList{mode: mode, vip: vip, maps: maps}
}

func (m MapTransferList) Operation() string { return MapTransferResultWriter }
func (m MapTransferList) String() string {
	return fmt.Sprintf("MapTransferList{mode=%d vip=%v maps=%v}", m.mode, m.vip, m.maps)
}

func (m MapTransferList) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteBool(m.vip)
		count := 5
		if m.vip {
			count = 10
		}
		for i := 0; i < count; i++ {
			v := _map.EmptyMapId
			if i < len(m.maps) {
				v = m.maps[i]
			}
			w.WriteInt(uint32(v))
		}
		return w.Bytes()
	}
}

// MapTransferError is the error form of MAP_TRANSFER_RESULT (modes 5-11):
// byte mode, byte targetList — no list payload.
type MapTransferError struct {
	mode byte
	vip  bool
}

func NewMapTransferError(mode byte, vip bool) MapTransferError {
	return MapTransferError{mode: mode, vip: vip}
}

func (m MapTransferError) Operation() string { return MapTransferResultWriter }
func (m MapTransferError) String() string {
	return fmt.Sprintf("MapTransferError{mode=%d vip=%v}", m.mode, m.vip)
}

func (m MapTransferError) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteBool(m.vip)
		return w.Bytes()
	}
}
