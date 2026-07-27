package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const TournamentSetPrizeWriter = "TournamentSetPrize"

// TournamentSetPrize mirrors CField_Tournament::OnTournamentSetPrize. Read
// order verified against the live IDBs and found identical in every version
// checked:
//
//	gms_v79 @0x5587e3, gms_v83 @0x57b815, gms_v87 @0x5a9f62,
//	gms_v95 @0x5633a0 (PDB-backed names), jms_v185 @0x5cffa7
//	(gms_v84 byte-identical to v83).
//
// Wire layout: Decode1(slot), Decode1(flag); flag!=0 branch Decode4s two
// item ids (both fed to CItemInfo::GetItemName, formatted into the
// "...PRIZE...1ST: %s...2ND: %s" client string — SP_917 in v83/v79). The
// flag==0 branch reads no further ints; slot instead selects one of two
// success/failure StringPool messages. The trailing client-side Delegate
// (sub_XXXXXX in the exports) is post-read application logic, not a wire
// read, and is excluded.
//
// The prior atlas codec wrote/read the two item ids unconditionally,
// silently desyncing the client whenever flag==0 — a false pass (the verify
// markers asserted the encoder's own four-field output, never the true
// gated wire body). Corrected here across all versions (task-181).
//
// packet-audit:fname CField_Tournament::OnTournamentSetPrize
type TournamentSetPrize struct {
	slot    byte
	flag    byte
	itemId1 uint32
	itemId2 uint32
}

func NewTournamentSetPrize(slot byte, flag byte, itemId1 uint32, itemId2 uint32) TournamentSetPrize {
	return TournamentSetPrize{slot: slot, flag: flag, itemId1: itemId1, itemId2: itemId2}
}

func (m TournamentSetPrize) Slot() byte      { return m.slot }
func (m TournamentSetPrize) Flag() byte      { return m.flag }
func (m TournamentSetPrize) ItemId1() uint32 { return m.itemId1 }
func (m TournamentSetPrize) ItemId2() uint32 { return m.itemId2 }

func (m TournamentSetPrize) Operation() string { return TournamentSetPrizeWriter }
func (m TournamentSetPrize) String() string {
	return fmt.Sprintf("slot [%d] flag [%d] itemId1 [%d] itemId2 [%d]", m.slot, m.flag, m.itemId1, m.itemId2)
}

// tournamentSetPrizeHasItems reports whether OnTournamentSetPrize reads the
// two item-id ints for the given flag byte. The client gates strictly on
// flag != 0; slot never participates in the gate.
func tournamentSetPrizeHasItems(flag byte) bool {
	return flag != 0
}

func (m TournamentSetPrize) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.slot)
		w.WriteByte(m.flag)
		if tournamentSetPrizeHasItems(m.flag) {
			w.WriteInt(m.itemId1)
			w.WriteInt(m.itemId2)
		}
		return w.Bytes()
	}
}

func (m *TournamentSetPrize) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.slot = r.ReadByte()
		m.flag = r.ReadByte()
		if tournamentSetPrizeHasItems(m.flag) {
			m.itemId1 = r.ReadUint32()
			m.itemId2 = r.ReadUint32()
		}
	}
}
