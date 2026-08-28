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
		return &PairRing{ // field order IS wire order; do not reorder
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
	r.Marriage = &MarriageRing{ // field order IS wire order; do not reorder
		MarriageCharacterId: rd.ReadUint32(),
		PartnerCharacterId:  rd.ReadUint32(),
		ItemId:              rd.ReadUint32(),
	}
}

// recordNameWidth is the wire width of every fixed name field in the record
// blocks below (GW_CoupleRecord::szPairCharacterName, GW_MarriageRecord::
// szGroomName / szBrideName). The client pushes the raw record bytes into a
// %s format argument with no length check, so a full 13-byte name with no
// NUL terminator over-reads; writeRecordName truncates the payload to 12
// bytes and always leaves at least one zero byte (task-269 3a derivation).
const recordNameWidth = 13

// writeRecordName writes a name truncated to recordNameWidth-1 bytes,
// zero-padded to the full recordNameWidth. This is deliberately NOT
// WritePaddedString: that helper truncates to the full width with no
// reserved terminator byte, which would over-read on the client for a
// maximum-length name.
func writeRecordName(w *response.Writer, name string) {
	b := []byte(name)
	if len(b) > recordNameWidth-1 {
		b = b[:recordNameWidth-1]
	}
	w.WriteByteArray(b)
	w.WriteByteArray(make([]byte, recordNameWidth-len(b)))
}

// readRecordName is the inverse of writeRecordName; ReadPaddedString already
// trims the trailing zero padding, so it is reused as-is.
func readRecordName(rd *request.Reader) string {
	return ReadPaddedString(rd, recordNameWidth)
}

// CoupleRecord is one entry of the couple-ring record block written by
// CharacterData (GW_CoupleRecord, 33 bytes / 0x21). Unlike PairRing (the
// avatar-block field codec above), this record carries no ring item id at
// all — design.md §2's split is refuted by the task-269 3a IDA derivation
// (docs/tasks/task-269-ring-pair-behavior/ring-field-derivation.md).
type CoupleRecord struct {
	PairCharacterId   uint32
	PairCharacterName string
	OwnSN             int64
	PairSN            int64
}

func (c CoupleRecord) encode(w *response.Writer) {
	w.WriteInt(c.PairCharacterId)
	writeRecordName(w, c.PairCharacterName)
	w.WriteInt64(c.OwnSN)
	w.WriteInt64(c.PairSN)
}

func decodeCoupleRecord(rd *request.Reader) CoupleRecord {
	return CoupleRecord{ // field order IS wire order; do not reorder
		PairCharacterId:   rd.ReadUint32(),
		PairCharacterName: readRecordName(rd),
		OwnSN:             rd.ReadInt64(),
		PairSN:            rd.ReadInt64(),
	}
}

// FriendRecord is one entry of the friendship-ring record block
// (GW_FriendRecord, 37 bytes / 0x25): the identical 33-byte CoupleRecord
// shape, plus the friendship ring's item template id.
type FriendRecord struct {
	CoupleRecord
	FriendItemId uint32
}

func (f FriendRecord) encode(w *response.Writer) {
	f.CoupleRecord.encode(w)
	w.WriteInt(f.FriendItemId)
}

func decodeFriendRecord(rd *request.Reader) FriendRecord {
	return FriendRecord{ // field order IS wire order; do not reorder
		CoupleRecord: decodeCoupleRecord(rd),
		FriendItemId: rd.ReadUint32(),
	}
}

// MarriageRecord is one entry of the marriage record block
// (GW_MarriageRecord, 48 bytes / 0x30). GroomId/BrideId is an unordered
// pair the client disambiguates by its own gender; this is NOT the same
// naming as MarriageRing.MarriageCharacterId/PartnerCharacterId above, which
// is the encoded character's own id versus the partner's (task-269 3a
// derivation) — the two record shapes are deliberately not unified.
type MarriageRecord struct {
	MarriageNo  uint32
	GroomId     uint32
	BrideId     uint32
	Status      uint16
	GroomItemId uint32
	BrideItemId uint32
	GroomName   string
	BrideName   string
}

func (m MarriageRecord) encode(w *response.Writer) {
	w.WriteInt(m.MarriageNo)
	w.WriteInt(m.GroomId)
	w.WriteInt(m.BrideId)
	w.WriteShort(m.Status)
	w.WriteInt(m.GroomItemId)
	w.WriteInt(m.BrideItemId)
	writeRecordName(w, m.GroomName)
	writeRecordName(w, m.BrideName)
}

func decodeMarriageRecord(rd *request.Reader) MarriageRecord {
	return MarriageRecord{ // field order IS wire order; do not reorder
		MarriageNo:  rd.ReadUint32(),
		GroomId:     rd.ReadUint32(),
		BrideId:     rd.ReadUint32(),
		Status:      rd.ReadUint16(),
		GroomItemId: rd.ReadUint32(),
		BrideItemId: rd.ReadUint32(),
		GroomName:   readRecordName(rd),
		BrideName:   readRecordName(rd),
	}
}

// RingRecords is the couple/friend/marriage record block written by site A
// (CharacterData), distinct from RingSet's avatar-block field codec above.
// Each arm is a count-prefixed (WriteShort) list of fixed-width records. A
// RingRecords with all three slices empty writes exactly the pre-task
// encoder's shape (PRD FR-9): three WriteShort(0) on a modern GMS/JMS
// variant, one WriteShort(0) on legacy GMS (<=28).
type RingRecords struct {
	Couple   []CoupleRecord
	Friend   []FriendRecord
	Marriage []MarriageRecord
}

// EncodeRecords writes the record block. The friend/marriage arms are gated
// verbatim off the pre-task encodeRings gate (character/data.go:767) so the
// v29..v60 shape of CharacterData does not move.
func (r RingRecords) EncodeRecords(w *response.Writer, t tenant.Model) {
	w.WriteShort(uint16(len(r.Couple)))
	for _, c := range r.Couple {
		c.encode(w)
	}

	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		w.WriteShort(uint16(len(r.Friend)))
		for _, f := range r.Friend {
			f.encode(w)
		}

		w.WriteShort(uint16(len(r.Marriage)))
		for _, m := range r.Marriage {
			m.encode(w)
		}
	}
}

// DecodeRecords is the inverse of EncodeRecords.
func (r *RingRecords) DecodeRecords(rd *request.Reader, t tenant.Model) {
	coupleCount := rd.ReadUint16()
	r.Couple = make([]CoupleRecord, coupleCount)
	for i := range r.Couple {
		r.Couple[i] = decodeCoupleRecord(rd)
	}

	if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
		friendCount := rd.ReadUint16()
		r.Friend = make([]FriendRecord, friendCount)
		for i := range r.Friend {
			r.Friend[i] = decodeFriendRecord(rd)
		}

		marriageCount := rd.ReadUint16()
		r.Marriage = make([]MarriageRecord, marriageCount)
		for i := range r.Marriage {
			r.Marriage[i] = decodeMarriageRecord(rd)
		}
	} else {
		r.Friend = nil
		r.Marriage = nil
	}
}
