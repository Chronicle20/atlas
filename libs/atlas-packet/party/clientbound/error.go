package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// --- OnPartyResult error/notice arms (discrete per mode; Task-2 AP split) -----
//
// The shared party/clientbound.Error catch-all is gone: every enumerated
// OnPartyResult error/notice arm now has its OWN discrete struct so it maps 1:1
// to a fixed operation key and a single dispatcher #-entry (mirrors the task-103
// guild OnGuildResult split). Mode bytes are resolved at emit time via the
// config "operations" table — NEVER literal in this file.
//
// Read orders verified case-by-case in the v83 OnPartyResult switch
// (@0xa3e31c) and confirmed against v84@0xa89cf3 / v87@0xad697a / v95@0xa10ab0 /
// jms@0xb297e7 (see docs/packets/dispatchers/party.yaml). Twelve arms read
// NOTHING after the dispatcher's Decode1(mode) — mode-only structs. Three
// invite-target arms (cases 21/22/23) each read a trailing DecodeStr(target)
// that is %s-substituted into a StringPool message — {mode,name} structs.
//
// D8 (IDA wins): UNABLE_TO_FIND_THE_CHARACTER (case 33) and
// UNABLE_TO_FIND_THE_REQUESTED_CHARACTER_IN_THIS_CHANNEL (case 19) read NO
// trailing DecodeStr in the switch, so they are MODE-ONLY here. The legacy
// shared Error wrote a trailing name for them — bytes the client never consumed.
// The migration intentionally drops those bytes (scoped to what the client reads).

// --- Mode-only arms -----------------------------------------------------------

// AlreadyJoined1 — ALREADY_HAVE_JOINED_A_PARTY_1 (case 9). Mode-only
// (v83 OnPartyResult@0xa3e31c case 9 → StringPool notice, no Decode*).
// packet-audit:fname CWvsContext::OnPartyResult#AlreadyJoined1
type AlreadyJoined1 struct{ mode byte }

func NewAlreadyJoined1(mode byte) AlreadyJoined1 { return AlreadyJoined1{mode: mode} }
func (m AlreadyJoined1) Operation() string       { return PartyOperationWriter }
func (m AlreadyJoined1) String() string          { return fmt.Sprintf("mode [%d]", m.mode) }
func (m AlreadyJoined1) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *AlreadyJoined1) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// BeginnerCannotCreate — A_BEGINNER_CANT_CREATE_A_PARTY (case 10). Mode-only
// (v83 OnPartyResult@0xa3e31c case 10 → StringPool notice, no Decode*).
// packet-audit:fname CWvsContext::OnPartyResult#BeginnerCannotCreate
type BeginnerCannotCreate struct{ mode byte }

func NewBeginnerCannotCreate(mode byte) BeginnerCannotCreate { return BeginnerCannotCreate{mode: mode} }
func (m BeginnerCannotCreate) Operation() string            { return PartyOperationWriter }
func (m BeginnerCannotCreate) String() string               { return fmt.Sprintf("mode [%d]", m.mode) }
func (m BeginnerCannotCreate) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *BeginnerCannotCreate) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// NotInParty — YOU_HAVE_YET_TO_JOIN_A_PARTY (case 13). Mode-only
// (v83 OnPartyResult@0xa3e31c case 13 → StringPool notice, no Decode*).
// packet-audit:fname CWvsContext::OnPartyResult#NotInParty
type NotInParty struct{ mode byte }

func NewNotInParty(mode byte) NotInParty { return NotInParty{mode: mode} }
func (m NotInParty) Operation() string   { return PartyOperationWriter }
func (m NotInParty) String() string      { return fmt.Sprintf("mode [%d]", m.mode) }
func (m NotInParty) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *NotInParty) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// AlreadyJoined2 — ALREADY_HAVE_JOINED_A_PARTY_2 (v83/v84 case 16; v87+ case 17).
// Mode-only (v83 OnPartyResult@0xa3e31c case 16 → StringPool notice, no Decode*).
// packet-audit:fname CWvsContext::OnPartyResult#AlreadyJoined2
type AlreadyJoined2 struct{ mode byte }

func NewAlreadyJoined2(mode byte) AlreadyJoined2 { return AlreadyJoined2{mode: mode} }
func (m AlreadyJoined2) Operation() string       { return PartyOperationWriter }
func (m AlreadyJoined2) String() string          { return fmt.Sprintf("mode [%d]", m.mode) }
func (m AlreadyJoined2) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *AlreadyJoined2) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// PartyFull — THE_PARTY_YOURE_TRYING_TO_JOIN_IS_ALREADY_IN_FULL_CAPACITY
// (v83/v84 case 17; v87+ case 18). Mode-only (v83 OnPartyResult@0xa3e31c case 17
// → StringPool notice, no Decode*).
// packet-audit:fname CWvsContext::OnPartyResult#PartyFull
type PartyFull struct{ mode byte }

func NewPartyFull(mode byte) PartyFull { return PartyFull{mode: mode} }
func (m PartyFull) Operation() string  { return PartyOperationWriter }
func (m PartyFull) String() string     { return fmt.Sprintf("mode [%d]", m.mode) }
func (m PartyFull) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *PartyFull) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// UnableToFindInChannel — UNABLE_TO_FIND_THE_REQUESTED_CHARACTER_IN_THIS_CHANNEL
// (case 19; v83/v84 only, version-absent v87+). Mode-only: the v83 switch case 19
// reads NO trailing DecodeStr (D8, IDA wins). The legacy Error wrote a name the
// client never consumed; the migration intentionally drops it.
// packet-audit:fname CWvsContext::OnPartyResult#UnableToFindInChannel
type UnableToFindInChannel struct{ mode byte }

func NewUnableToFindInChannel(mode byte) UnableToFindInChannel {
	return UnableToFindInChannel{mode: mode}
}
func (m UnableToFindInChannel) Operation() string { return PartyOperationWriter }
func (m UnableToFindInChannel) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m UnableToFindInChannel) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *UnableToFindInChannel) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// CannotKick — CANNOT_KICK_ANOTHER_USER_IN_THIS_MAP (v83/v84 case 25; v87+ case 29).
// Mode-only (v83 OnPartyResult@0xa3e31c case 25 → StringPool notice, no Decode*).
// packet-audit:fname CWvsContext::OnPartyResult#CannotKick
type CannotKick struct{ mode byte }

func NewCannotKick(mode byte) CannotKick { return CannotKick{mode: mode} }
func (m CannotKick) Operation() string   { return PartyOperationWriter }
func (m CannotKick) String() string      { return fmt.Sprintf("mode [%d]", m.mode) }
func (m CannotKick) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *CannotKick) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// OnlyWithinVicinity — THIS_CAN_ONLY_BE_GIVEN_TO_A_PARTY_MEMBER_WITHIN_THE_VICINITY
// (v83/v84 case 28; v87+ case 32). Mode-only (v83 OnPartyResult@0xa3e31c case 28
// → StringPool notice, no Decode*).
// packet-audit:fname CWvsContext::OnPartyResult#OnlyWithinVicinity
type OnlyWithinVicinity struct{ mode byte }

func NewOnlyWithinVicinity(mode byte) OnlyWithinVicinity { return OnlyWithinVicinity{mode: mode} }
func (m OnlyWithinVicinity) Operation() string          { return PartyOperationWriter }
func (m OnlyWithinVicinity) String() string             { return fmt.Sprintf("mode [%d]", m.mode) }
func (m OnlyWithinVicinity) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *OnlyWithinVicinity) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// UnableToHandOver — UNABLE_TO_HAND_OVER_THE_LEADERSHIP_POST_NO_PARTY_MEMBER…
// (v83/v84 case 29; v87+ case 33). Mode-only (v83 OnPartyResult@0xa3e31c case 29
// → StringPool notice, no Decode*).
// packet-audit:fname CWvsContext::OnPartyResult#UnableToHandOver
type UnableToHandOver struct{ mode byte }

func NewUnableToHandOver(mode byte) UnableToHandOver { return UnableToHandOver{mode: mode} }
func (m UnableToHandOver) Operation() string         { return PartyOperationWriter }
func (m UnableToHandOver) String() string            { return fmt.Sprintf("mode [%d]", m.mode) }
func (m UnableToHandOver) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *UnableToHandOver) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// OnlySameChannel — YOU_MAY_ONLY_CHANGE_WITH_THE_PARTY_MEMBER_THATS_ON_THE_SAME_CHANNEL
// (v83/v84 case 30; v87+ case 34). Mode-only (v83 OnPartyResult@0xa3e31c case 30
// → StringPool notice, no Decode*).
// packet-audit:fname CWvsContext::OnPartyResult#OnlySameChannel
type OnlySameChannel struct{ mode byte }

func NewOnlySameChannel(mode byte) OnlySameChannel { return OnlySameChannel{mode: mode} }
func (m OnlySameChannel) Operation() string        { return PartyOperationWriter }
func (m OnlySameChannel) String() string           { return fmt.Sprintf("mode [%d]", m.mode) }
func (m OnlySameChannel) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *OnlySameChannel) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// GmCannotCreate — AS_A_GM_YOURE_FORBIDDEN_FROM_CREATING_A_PARTY (v83/v84 case 32;
// v87+ case 36). Mode-only (v83 OnPartyResult@0xa3e31c case 32 → StringPool
// notice, no Decode*).
// packet-audit:fname CWvsContext::OnPartyResult#GmCannotCreate
type GmCannotCreate struct{ mode byte }

func NewGmCannotCreate(mode byte) GmCannotCreate { return GmCannotCreate{mode: mode} }
func (m GmCannotCreate) Operation() string       { return PartyOperationWriter }
func (m GmCannotCreate) String() string          { return fmt.Sprintf("mode [%d]", m.mode) }
func (m GmCannotCreate) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *GmCannotCreate) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// UnableToFindCharacter — UNABLE_TO_FIND_THE_CHARACTER (v83/v84 case 33; v87/v95
// case 37; version-absent jms). Mode-only: the v83 switch case 33 reads NO
// trailing DecodeStr (D8, IDA wins). The legacy Error wrote a name the client
// never consumed; the migration intentionally drops it.
// packet-audit:fname CWvsContext::OnPartyResult#UnableToFindCharacter
type UnableToFindCharacter struct{ mode byte }

func NewUnableToFindCharacter(mode byte) UnableToFindCharacter {
	return UnableToFindCharacter{mode: mode}
}
func (m UnableToFindCharacter) Operation() string { return PartyOperationWriter }
func (m UnableToFindCharacter) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m UnableToFindCharacter) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *UnableToFindCharacter) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// --- Invite-target arms (discrete per mode; {mode,name}) ----------------------
//
// These three arms each read a trailing player-name string from the wire and
// %s-substitute it into a StringPool message. Verified case-by-case in the v83
// OnPartyResult switch (@0xa3e31c): cases 21/22/23 each do CInPacket::DecodeStr
// then ZXString::Format("%s", wireString). v83/v84 only (version-absent v87+).

// BlockingInvitations — IS_CURRENTLY_BLOCKING_ANY_PARTY_INVITATIONS (case 21;
// v83/v84 only). Decode1(mode) + DecodeStr(name). v83 OnPartyResult@0xa3e31c case 21.
// packet-audit:fname CWvsContext::OnPartyResult#BlockingInvitations
type BlockingInvitations struct {
	mode byte
	name string
}

func NewBlockingInvitations(mode byte, name string) BlockingInvitations {
	return BlockingInvitations{mode: mode, name: name}
}
func (m BlockingInvitations) Operation() string { return PartyOperationWriter }
func (m BlockingInvitations) String() string {
	return fmt.Sprintf("mode [%d], name [%s]", m.mode, m.name)
}
func (m BlockingInvitations) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.name) // DecodeStr target name (case 21)
		return w.Bytes()
	}
}
func (m *BlockingInvitations) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		m.mode = r.ReadByte()
		m.name = r.ReadAsciiString()
	}
}

// TakingCareOfInvitation — IS_TAKING_CARE_OF_ANOTHER_INVITATION (case 22;
// v83/v84 only). Decode1(mode) + DecodeStr(name). v83 OnPartyResult@0xa3e31c case 22.
// packet-audit:fname CWvsContext::OnPartyResult#TakingCareOfInvitation
type TakingCareOfInvitation struct {
	mode byte
	name string
}

func NewTakingCareOfInvitation(mode byte, name string) TakingCareOfInvitation {
	return TakingCareOfInvitation{mode: mode, name: name}
}
func (m TakingCareOfInvitation) Operation() string { return PartyOperationWriter }
func (m TakingCareOfInvitation) String() string {
	return fmt.Sprintf("mode [%d], name [%s]", m.mode, m.name)
}
func (m TakingCareOfInvitation) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.name) // DecodeStr target name (case 22)
		return w.Bytes()
	}
}
func (m *TakingCareOfInvitation) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		m.mode = r.ReadByte()
		m.name = r.ReadAsciiString()
	}
}

// RequestDenied — HAVE_DENIED_REQUEST_TO_THE_PARTY (case 23; v83/v84 only).
// Decode1(mode) + DecodeStr(name). v83 OnPartyResult@0xa3e31c case 23.
// packet-audit:fname CWvsContext::OnPartyResult#RequestDenied
type RequestDenied struct {
	mode byte
	name string
}

func NewRequestDenied(mode byte, name string) RequestDenied {
	return RequestDenied{mode: mode, name: name}
}
func (m RequestDenied) Operation() string { return PartyOperationWriter }
func (m RequestDenied) String() string {
	return fmt.Sprintf("mode [%d], name [%s]", m.mode, m.name)
}
func (m RequestDenied) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.name) // DecodeStr target name (case 23)
		return w.Bytes()
	}
}
func (m *RequestDenied) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		m.mode = r.ReadByte()
		m.name = r.ReadAsciiString()
	}
}
