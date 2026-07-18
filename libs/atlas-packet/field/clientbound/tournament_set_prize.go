package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const TournamentSetPrizeWriter = "TournamentSetPrize"

// TournamentSetPrize mirrors CField_Tournament::OnTournamentSetPrize. The wire
// shape is two leading bytes (Decode1) followed by two ints (Decode4). The
// trailing client-side Delegate (sub_XXXXXX in the v83/v87/jms exports) is
// post-read application logic, not a wire read, and is excluded.
// packet-audit:fname CField_Tournament::OnTournamentSetPrize
type TournamentSetPrize struct {
	slot   byte
	flag   byte
	itemId uint32
	count  uint32
}

func NewTournamentSetPrize(slot byte, flag byte, itemId uint32, count uint32) TournamentSetPrize {
	return TournamentSetPrize{slot: slot, flag: flag, itemId: itemId, count: count}
}

func (m TournamentSetPrize) Slot() byte     { return m.slot }
func (m TournamentSetPrize) Flag() byte     { return m.flag }
func (m TournamentSetPrize) ItemId() uint32 { return m.itemId }
func (m TournamentSetPrize) Count() uint32  { return m.count }

func (m TournamentSetPrize) Operation() string { return TournamentSetPrizeWriter }
func (m TournamentSetPrize) String() string {
	return fmt.Sprintf("slot [%d] flag [%d] itemId [%d] count [%d]", m.slot, m.flag, m.itemId, m.count)
}

func (m TournamentSetPrize) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.slot)
		w.WriteByte(m.flag)
		w.WriteInt(m.itemId)
		w.WriteInt(m.count)
		return w.Bytes()
	}
}

func (m *TournamentSetPrize) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.slot = r.ReadByte()
		m.flag = r.ReadByte()
		m.itemId = r.ReadUint32()
		m.count = r.ReadUint32()
	}
}
