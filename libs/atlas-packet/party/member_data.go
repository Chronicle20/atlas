package party

import (
	"context"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type PartyMember struct {
	Id        uint32
	Name      string
	JobId     uint16
	Level     uint16
	ChannelId int32 // -2 if offline
	MapId     uint32
}

// WritePartyData serialises PARTYDATA to the wire. The v83 PARTYDATA struct is
// 298 bytes (6×portal = TownID+FieldID+x+y, no skillId, no PQ fields). v95
// extended portals to 5 fields (+skillId = 24 bytes) and added PQ reward
// arrays (56 bytes), bringing the total to 378 bytes. These additions are
// gated on GMS MajorVersion >= 95 only (confirmed via IDA):
//   v83 OnPartyResult@0xa3e31c: memset(3732,0,0x12A=298)
//   v95 OnPartyResult: memset(3732,0,0x17A=378)
//   JMS v185 OnPartyResult@0xb297e7: qmemcpy(v120,...,0x12Au=298) — JMS uses the
//   small PARTYDATA (same as v83), so JMS is explicitly excluded from v95plus.
func WritePartyData(ctx context.Context, w *response.Writer, members []PartyMember, leaderId uint32) {
	t := tenant.MustFromContext(ctx)
	for _, m := range members {
		w.WriteInt(m.Id)
	}
	for range 6 - len(members) {
		w.WriteInt(0)
	}
	for _, m := range members {
		model.WritePaddedString(w, m.Name, 13)
	}
	for range 6 - len(members) {
		model.WritePaddedString(w, "", 13)
	}
	for _, m := range members {
		w.WriteInt(uint32(m.JobId))
	}
	for range 6 - len(members) {
		w.WriteInt(0)
	}
	for _, m := range members {
		w.WriteInt(uint32(m.Level))
	}
	for range 6 - len(members) {
		w.WriteInt(0)
	}
	for _, m := range members {
		w.WriteInt32(m.ChannelId)
	}
	for range 6 - len(members) {
		w.WriteInt(0)
	}
	w.WriteInt(leaderId)
	for _, m := range members {
		w.WriteInt(m.MapId)
	}
	for range 6 - len(members) {
		w.WriteInt(0)
	}
	// aTownPortal[6]: v83 = 4 ints (town+field+x+y); v95+ adds m_nSKillID (5th int).
	// JMS v185 uses the small PARTYDATA (0x12A=298 bytes), same as GMS v83.
	// IDA evidence: JMS OnPartyResult@0xb297e7 qmemcpy(v120,...,0x12Au=298).
	v95plus := t.Region() == "GMS" && t.MajorVersion() >= 95
	for range 6 {
		w.WriteInt(uint32(_map.EmptyMapId)) // m_dwTownID
		w.WriteInt(uint32(_map.EmptyMapId)) // m_dwFieldID
		if v95plus {
			w.WriteInt(0) // m_nSKillID (v95+)
		}
		w.WriteInt(0) // m_ptFieldPortal.x
		w.WriteInt(0) // m_ptFieldPortal.y
	}
	// aPQReward[6], aPQRewardType[6], dwPQRewardMobTemplateID, bPQReward (v95+).
	// Absent in v83 (memset size 0x12A=298 vs v95 0x17A=378).
	if v95plus {
		for range 6 {
			w.WriteInt(0) // aPQReward[i]
		}
		for range 6 {
			w.WriteInt(0) // aPQRewardType[i]
		}
		w.WriteInt(0) // dwPQRewardMobTemplateID
		w.WriteInt(0) // bPQReward
	}
}

func ReadPartyData(ctx context.Context, r *request.Reader) ([]PartyMember, uint32) {
	t := tenant.MustFromContext(ctx)
	// JMS v185 uses the small PARTYDATA (0x12A=298 bytes), same as GMS v83.
	// IDA evidence: JMS OnPartyResult@0xb297e7 qmemcpy(v120,...,0x12Au=298).
	v95plus := t.Region() == "GMS" && t.MajorVersion() >= 95
	ids := make([]uint32, 6)
	for i := range 6 {
		ids[i] = r.ReadUint32()
	}
	names := make([]string, 6)
	for i := range 6 {
		names[i] = model.ReadPaddedString(r, 13)
	}
	jobs := make([]uint16, 6)
	for i := range 6 {
		jobs[i] = uint16(r.ReadUint32())
	}
	levels := make([]uint16, 6)
	for i := range 6 {
		levels[i] = uint16(r.ReadUint32())
	}
	channels := make([]int32, 6)
	for i := range 6 {
		channels[i] = r.ReadInt32()
	}
	leaderId := r.ReadUint32()
	maps := make([]uint32, 6)
	for i := range 6 {
		maps[i] = r.ReadUint32()
	}
	for range 6 {
		_ = r.ReadUint32() // m_dwTownID
		_ = r.ReadUint32() // m_dwFieldID
		if v95plus {
			_ = r.ReadUint32() // m_nSKillID (v95+)
		}
		_ = r.ReadUint32() // m_ptFieldPortal.x
		_ = r.ReadUint32() // m_ptFieldPortal.y
	}
	if v95plus {
		for range 6 {
			_ = r.ReadUint32() // aPQReward[i]
		}
		for range 6 {
			_ = r.ReadUint32() // aPQRewardType[i]
		}
		_ = r.ReadUint32() // dwPQRewardMobTemplateID
		_ = r.ReadUint32() // bPQReward
	}
	var members []PartyMember
	for i := range 6 {
		if ids[i] != 0 {
			members = append(members, PartyMember{
				Id:        ids[i],
				Name:      names[i],
				JobId:     jobs[i],
				Level:     levels[i],
				ChannelId: channels[i],
				MapId:     maps[i],
			})
		}
	}
	return members, leaderId
}
