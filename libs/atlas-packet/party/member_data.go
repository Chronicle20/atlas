package party

import (
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

type PartyMember struct {
	Id        uint32
	Name      string
	JobId     uint16
	Level     uint16
	ChannelId int32 // -2 if offline
	MapId     uint32
}

func WritePartyData(w *response.Writer, members []PartyMember, leaderId uint32) {
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
	for range 6 {
		w.WriteInt(uint32(_map.EmptyMapId)) // m_dwTownID
		w.WriteInt(uint32(_map.EmptyMapId)) // m_dwFieldID
		w.WriteInt(0)                       // m_nSKillID
		w.WriteInt(0)                       // m_ptFieldPortal.x
		w.WriteInt(0)                       // m_ptFieldPortal.y
	}
	// aPQReward[6], aPQRewardType[6], dwPQRewardMobTemplateID, bPQReward
	for range 6 {
		w.WriteInt(0) // aPQReward[i]
	}
	for range 6 {
		w.WriteInt(0) // aPQRewardType[i]
	}
	w.WriteInt(0) // dwPQRewardMobTemplateID
	w.WriteInt(0) // bPQReward
}

func ReadPartyData(r *request.Reader) ([]PartyMember, uint32) {
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
		_ = r.ReadUint32() // m_nSKillID
		_ = r.ReadUint32() // m_ptFieldPortal.x
		_ = r.ReadUint32() // m_ptFieldPortal.y
	}
	for range 6 {
		_ = r.ReadUint32() // aPQReward[i]
	}
	for range 6 {
		_ = r.ReadUint32() // aPQRewardType[i]
	}
	_ = r.ReadUint32() // dwPQRewardMobTemplateID
	_ = r.ReadUint32() // bPQReward
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
