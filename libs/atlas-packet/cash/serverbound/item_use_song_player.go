package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// packet-audit:fname CWvsContext::SendConsumeCashItemUseRequest
//
// ItemUseSongPlayer is the USE_CASH_ITEM sub-body for a song player (jukebox)
// cash item, item classification 510 — cash-slot type 20 on every version
// examined (get_cashslot_item_type @0x488c70 on GMS v95.0: `case 510: result = 20`).
//
// The sub-body is exactly one int32: the WZ sound's own IWzSound::length, in
// milliseconds. IDA-verified on two builds that bracket the supported range:
//
//	GMS v95.0 (GMS_v95.0_U_DEVM.exe) case-20 arm @0x9ed51e — reads the item's
//	  info/path node (StringPool 0x734), resolves it via IWzResMan::GetObjectA
//	  @0x9ed75a, casts to IWzSound @0x9ed773, calls IWzSound::Getlength
//	  @0x9ed7af and passes the result straight to COutPacket::Encode4 @0x9ed7b9.
//	GMS v83 (MapleStory_dump.exe) case-20 arm @0xa0c1a2 — identical sequence,
//	  Getlength via the vtable+56 getter sub_644DCF @0xa0c3ed then Encode4
//	  @0xa0c3f6.
//
// Exactly one Encode4 in the arm on both.
//
// The trailing updateTime is NOT part of this arm: it comes from the shared
// send tail on the versions that trail it (GMS <= v84), exactly as documented
// on ItemUseMorphCoupon. cashsb.UpdateTimeFirst(tenant) selects which.
//
// The server never resolves the BGM. The client reads the item's own
// info/path node in CMapLoadable::PlayNextMusic @0x61dab0 and hands it to
// CSoundMan::PlayBGM, so no BGM name crosses the wire in either direction.
type ItemUseSongPlayer struct {
	soundLengthMs   uint32
	updateTime      uint32
	updateTimeFirst bool
}

func NewItemUseSongPlayer(updateTimeFirst bool) *ItemUseSongPlayer {
	return &ItemUseSongPlayer{updateTimeFirst: updateTimeFirst}
}

func (m ItemUseSongPlayer) SoundLengthMs() uint32 { return m.soundLengthMs }
func (m ItemUseSongPlayer) UpdateTime() uint32    { return m.updateTime }

func (m ItemUseSongPlayer) Operation() string { return "ItemUseSongPlayer" }

func (m ItemUseSongPlayer) String() string {
	return fmt.Sprintf("soundLengthMs [%d] updateTime [%d]", m.soundLengthMs, m.updateTime)
}

func (m ItemUseSongPlayer) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.soundLengthMs)
		if !m.updateTimeFirst {
			w.WriteInt(m.updateTime)
		}
		return w.Bytes()
	}
}

func (m *ItemUseSongPlayer) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.soundLengthMs = r.ReadUint32()
		if !m.updateTimeFirst {
			m.updateTime = r.ReadUint32()
		}
	}
}
