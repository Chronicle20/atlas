package model

import (
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// PairRing is a couple or friendship ring pair: the character's own ring
// item serial number, the partner's ring item serial number, and the ring
// item's template id.
//
// The trailing field is the ring's item template id, NOT a pair character
// id (task-269 Task 1 verdict, docs/tasks/task-269-ring-pair-behavior/
// ring-field-derivation.md). It is passed straight into the int nItemID
// parameter of CUserPool::On{Couple,Friend}RecordAdd (v95 @0x94d600 /
// @0x94d700) and consumed by a WZ effect-layer load
// (CUser::SetCoupleItemEffect @0x8f05d0); the client derives the pair
// character id itself by searching the user pool on the two SNs
// (@0x94d637 / @0x94d675). The checked-in exports gms_v83.json / gms_v87.json
// / gms_v95.json call this field dwPairCharacterId / dwFriendCharacterId;
// that comment is wrong and must not be inherited.
type PairRing struct {
	OwnSN     int64
	PartnerSN int64
	ItemId    uint32
}

// MarriageRing is the marriage ring arm: the encoded character's own
// character id, the partner's character id, and the wedding ring item's
// template id.
//
// MarriageCharacterId is the encoded character's OWN character id, not the
// partner's (task-269 Task 1 verdict). design.md §3.1 named this field
// MarriageId; that name is refuted by v95 CUserPool::OnMarriageRecordAdd
// @0x94d800, which matches the field against a DIFFERENT user's
// m_dwMarriagePairCharacterID, and by CUserLocal::SetMarriagePairCharacterID
// @0x9089b0, which fills it from the local character's own bride/groom id
// (GW_MarriageRecord::dwBrideID / dwGroomID).
type MarriageRing struct {
	MarriageCharacterId uint32
	PartnerCharacterId  uint32
	ItemId              uint32
}

// RingSet is the couple/friendship/marriage ring block shared by spawn
// (CUserRemote::Init) and avatar-update (CUserRemote::OnAvatarModified). A
// nil pointer is the "no ring" signal for that arm.
type RingSet struct {
	Couple     *PairRing
	Friendship *PairRing
	Marriage   *MarriageRing
}

// EncodeField writes the ring block: bCouple flag [+ couple pair], bFriendship
// flag [+ friendship pair], bMarriage flag [+ marriage triple]. A RingSet with
// all three arms nil writes exactly three zero flag bytes, byte-identical to
// the pre-task encoder (PRD FR-9).
//
// The only branch is on region (JMS vs GMS): JMS wraps each pair arm's SN/item
// payload in an Int entry count (gms_jms_185.json CUserRemote::Init @0xa52876
// calls 34-41; count is 1 for a populated ring, and the whole per-entry
// payload is omitted, not zero-counted, when the arm is nil). There is
// deliberately no version gate inside the GMS arm: v48
// CUserPool::OnUserEnterField @0x6b277b calls 49-60 and v95
// CUserRemote::OnAvatarModified @0x954110 write the identical shape, so a
// later reader must not "fix" this by adding one (design.md §2 OQ-1). The
// marriage arm (flag + 3 Ints) is identical for GMS and JMS.
func (r RingSet) EncodeField(w *response.Writer, t tenant.Model) {
	isJMS := t.Region() == "JMS"

	encodePair := func(p *PairRing) {
		if p == nil {
			w.WriteByte(0)
			return
		}
		w.WriteByte(1)
		if isJMS {
			w.WriteInt(1) // entry count; always 1, this codec carries a single pair per arm
		}
		w.WriteInt64(p.OwnSN)
		w.WriteInt64(p.PartnerSN)
		w.WriteInt(p.ItemId)
	}

	encodePair(r.Couple)
	encodePair(r.Friendship)

	if r.Marriage == nil {
		w.WriteByte(0)
		return
	}
	w.WriteByte(1)
	w.WriteInt(r.Marriage.MarriageCharacterId)
	w.WriteInt(r.Marriage.PartnerCharacterId)
	w.WriteInt(r.Marriage.ItemId)
}

// DecodeField is the inverse of EncodeField.
func (r *RingSet) DecodeField(rd *request.Reader, t tenant.Model) {
	isJMS := t.Region() == "JMS"

	decodePair := func() *PairRing {
		if rd.ReadByte() == 0 {
			return nil
		}
		if isJMS {
			_ = rd.ReadUint32() // entry count; always 1, this codec carries a single pair per arm
		}
		return &PairRing{
			OwnSN:     rd.ReadInt64(),
			PartnerSN: rd.ReadInt64(),
			ItemId:    rd.ReadUint32(),
		}
	}

	r.Couple = decodePair()
	r.Friendship = decodePair()

	if rd.ReadByte() == 0 {
		r.Marriage = nil
		return
	}
	r.Marriage = &MarriageRing{
		MarriageCharacterId: rd.ReadUint32(),
		PartnerCharacterId:  rd.ReadUint32(),
		ItemId:              rd.ReadUint32(),
	}
}
