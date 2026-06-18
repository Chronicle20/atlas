package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

const GuildOperationWriter = "GuildOperation"

// RequestAgreement — REQUEST_AGREEMENT (case 3): the create-guild-agree dialog
// REQUEST broadcast to party members. The single clientbound mode here (F5); the
// serverbound member reply is the separate guild/serverbound AgreementResponse.
// packet-audit:fname CWvsContext::OnGuildResult#RequestAgreement
type RequestAgreement struct {
	mode       byte
	partyId    uint32
	leaderName string
	guildName  string
}

func NewRequestAgreement(mode byte, partyId uint32, leaderName string, guildName string) RequestAgreement {
	return RequestAgreement{mode: mode, partyId: partyId, leaderName: leaderName, guildName: guildName}
}

func (m RequestAgreement) Operation() string { return GuildOperationWriter }
func (m RequestAgreement) String() string {
	return fmt.Sprintf("mode [%d], partyId [%d], leaderName [%s], guildName [%s]", m.mode, m.partyId, m.leaderName, m.guildName)
}

func (m RequestAgreement) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.partyId)
		w.WriteAsciiString(m.leaderName)
		w.WriteAsciiString(m.guildName)
		return w.Bytes()
	}
}

func (m *RequestAgreement) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.partyId = r.ReadUint32()
		m.leaderName = r.ReadAsciiString()
		m.guildName = r.ReadAsciiString()
	}
}

// --- Mode-only error/notice arms (discrete per mode; AP-1 split) -------------
//
// Each arm's CWvsContext::OnGuildResult sub-handler reads NOTHING after the
// dispatcher's Decode1(mode) — it shows a StringPool / GuildNPCSay / CHATLOG
// message locally and returns. Each mode therefore has its OWN discrete struct
// that writes exactly that one mode byte (the byte its config-resolved body
// function receives) — no shared catch-all (task-103: discrete-per-mode rule;
// replaces the retired ErrorMessage struct).
//
// Read orders verified case-by-case in the v83 OnGuildResult switch
// (sub @0xa37490): every case below has an EMPTY decode list (no Decode* after
// the mode byte). v83/v84/v87/jms are byte-identical (mode bytes only differ on
// v95, the non-uniform shift; per-version bytes live in guild.yaml). Mode bytes
// are resolved at emit time, NEVER literal in this file.

// modeOnlyArm is the shared field/behaviour set for a discrete mode-only arm.
// Each named arm below is its OWN type (discrete-per-mode) so it maps 1:1 to a
// fixed operation key and a single dispatcher #-entry; the field shape is
// identical (one mode byte) but the types are distinct on purpose.

// RequestName — REQUEST_NAME (case 0x01). Prompts the create-guild name
// dialog; the sub-handler reads no body. v83 OnGuildResult@0xa37490 if-chain
// (v4==1 → CField::InputGuildName, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#RequestName
type RequestName struct{ mode byte }

func NewRequestName(mode byte) RequestName { return RequestName{mode: mode} }
func (m RequestName) Operation() string    { return GuildOperationWriter }
func (m RequestName) String() string       { return fmt.Sprintf("mode [%d]", m.mode) }
func (m RequestName) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *RequestName) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// RequestEmblem — REQUEST_EMBLEM (case 0x11). Prompts the set-emblem
// dialog; no body. v83 OnGuildResult@0xa37490 (v4==0x11 → CSetGuildMarkDlg, no
// Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#RequestEmblem
type RequestEmblem struct{ mode byte }

func NewRequestEmblem(mode byte) RequestEmblem { return RequestEmblem{mode: mode} }
func (m RequestEmblem) Operation() string      { return GuildOperationWriter }
func (m RequestEmblem) String() string         { return fmt.Sprintf("mode [%d]", m.mode) }
func (m RequestEmblem) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *RequestEmblem) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// CreateErrorNameInUse — THE_NAME_IS_ALREADY_IN_USE (case 0x1C). Mode-only
// (v83 OnGuildResult@0xa37490, v4==0x1C → StringPool notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#CreateErrorNameInUse
type CreateErrorNameInUse struct{ mode byte }

func NewCreateErrorNameInUse(mode byte) CreateErrorNameInUse {
	return CreateErrorNameInUse{mode: mode}
}
func (m CreateErrorNameInUse) Operation() string { return GuildOperationWriter }
func (m CreateErrorNameInUse) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m CreateErrorNameInUse) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *CreateErrorNameInUse) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// CreateErrorDisagreed — SOMEBODY_HAS_DISAGREED (case 0x24). Mode-only
// (v83 OnGuildResult@0xa37490, case 0x24 → notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#CreateErrorDisagreed
type CreateErrorDisagreed struct{ mode byte }

func NewCreateErrorDisagreed(mode byte) CreateErrorDisagreed {
	return CreateErrorDisagreed{mode: mode}
}
func (m CreateErrorDisagreed) Operation() string { return GuildOperationWriter }
func (m CreateErrorDisagreed) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m CreateErrorDisagreed) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *CreateErrorDisagreed) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// CreateError — THE_PROBLEM…FORMING_THE_GUILD (case 0x26). Mode-only
// (v83 OnGuildResult@0xa37490, case 0x26 → notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#CreateError
type CreateError struct{ mode byte }

func NewCreateError(mode byte) CreateError { return CreateError{mode: mode} }
func (m CreateError) Operation() string    { return GuildOperationWriter }
func (m CreateError) String() string       { return fmt.Sprintf("mode [%d]", m.mode) }
func (m CreateError) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *CreateError) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// JoinErrorAlreadyJoined — ALREADY_JOINED_THE_GUILD (case 0x28). Mode-only
// (v83 OnGuildResult@0xa37490, case 0x28 → notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#JoinErrorAlreadyJoined
type JoinErrorAlreadyJoined struct{ mode byte }

func NewJoinErrorAlreadyJoined(mode byte) JoinErrorAlreadyJoined {
	return JoinErrorAlreadyJoined{mode: mode}
}
func (m JoinErrorAlreadyJoined) Operation() string { return GuildOperationWriter }
func (m JoinErrorAlreadyJoined) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m JoinErrorAlreadyJoined) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *JoinErrorAlreadyJoined) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// JoinErrorMaxMembers — MAX_NUMBER_OF_USERS (case 0x29). Mode-only
// (v83 OnGuildResult@0xa37490, case 0x29 → notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#JoinErrorMaxMembers
type JoinErrorMaxMembers struct{ mode byte }

func NewJoinErrorMaxMembers(mode byte) JoinErrorMaxMembers {
	return JoinErrorMaxMembers{mode: mode}
}
func (m JoinErrorMaxMembers) Operation() string { return GuildOperationWriter }
func (m JoinErrorMaxMembers) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m JoinErrorMaxMembers) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *JoinErrorMaxMembers) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// JoinErrorNotInChannel — CHARACTER_CANNOT_BE_FOUND_IN_THE_CURRENT_CHANNEL
// (case 0x2A). Mode-only (v83 OnGuildResult@0xa37490, case 0x2A → notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#JoinErrorNotInChannel
type JoinErrorNotInChannel struct{ mode byte }

func NewJoinErrorNotInChannel(mode byte) JoinErrorNotInChannel {
	return JoinErrorNotInChannel{mode: mode}
}
func (m JoinErrorNotInChannel) Operation() string { return GuildOperationWriter }
func (m JoinErrorNotInChannel) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m JoinErrorNotInChannel) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *JoinErrorNotInChannel) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// MemberQuitErrorNotInGuild — MEMBER_QUIT_ERROR_NOT_IN_GUILD (case 0x2D).
// Mode-only (v83 OnGuildResult@0xa37490, case 0x2D → notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#MemberQuitErrorNotInGuild
type MemberQuitErrorNotInGuild struct{ mode byte }

func NewMemberQuitErrorNotInGuild(mode byte) MemberQuitErrorNotInGuild {
	return MemberQuitErrorNotInGuild{mode: mode}
}
func (m MemberQuitErrorNotInGuild) Operation() string { return GuildOperationWriter }
func (m MemberQuitErrorNotInGuild) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m MemberQuitErrorNotInGuild) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *MemberQuitErrorNotInGuild) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// MemberExpelledErrorNotInGuild — MEMBER_EXPELLED_ERROR_NOT_IN_GUILD
// (case 0x30). Mode-only (v83 OnGuildResult@0xa37490, case 0x30 → notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#MemberExpelledErrorNotInGuild
type MemberExpelledErrorNotInGuild struct{ mode byte }

func NewMemberExpelledErrorNotInGuild(mode byte) MemberExpelledErrorNotInGuild {
	return MemberExpelledErrorNotInGuild{mode: mode}
}
func (m MemberExpelledErrorNotInGuild) Operation() string { return GuildOperationWriter }
func (m MemberExpelledErrorNotInGuild) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m MemberExpelledErrorNotInGuild) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *MemberExpelledErrorNotInGuild) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// DisbandError — THE_PROBLEM…DISBANDING_THE_GUILD (case 0x34). Mode-only
// (v83 OnGuildResult@0xa37490, case 0x34 → GuildNPCSay, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#DisbandError
type DisbandError struct{ mode byte }

func NewDisbandError(mode byte) DisbandError { return DisbandError{mode: mode} }
func (m DisbandError) Operation() string     { return GuildOperationWriter }
func (m DisbandError) String() string        { return fmt.Sprintf("mode [%d]", m.mode) }
func (m DisbandError) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *DisbandError) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// CreateErrorCannotAsAdmin — ADMIN_CANNOT_MAKE_A_GUILD (case 0x38).
// Mode-only (v83 OnGuildResult@0xa37490, case 0x38 → CHATLOG, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#CreateErrorCannotAsAdmin
type CreateErrorCannotAsAdmin struct{ mode byte }

func NewCreateErrorCannotAsAdmin(mode byte) CreateErrorCannotAsAdmin {
	return CreateErrorCannotAsAdmin{mode: mode}
}
func (m CreateErrorCannotAsAdmin) Operation() string { return GuildOperationWriter }
func (m CreateErrorCannotAsAdmin) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m CreateErrorCannotAsAdmin) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *CreateErrorCannotAsAdmin) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// IncreaseCapacityError — THE_PROBLEM…INCREASING_THE_GUILD (case 0x3B).
// Mode-only (v83 OnGuildResult@0xa37490, case 0x3B → notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#IncreaseCapacityError
type IncreaseCapacityError struct{ mode byte }

func NewIncreaseCapacityError(mode byte) IncreaseCapacityError {
	return IncreaseCapacityError{mode: mode}
}
func (m IncreaseCapacityError) Operation() string { return GuildOperationWriter }
func (m IncreaseCapacityError) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m IncreaseCapacityError) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *IncreaseCapacityError) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// QuestErrorLessThanSixMembers — THERE_ARE_LESS_THAN_6_MEMBERS (case 0x4A).
// Mode-only (v83 OnGuildResult@0xa37490, case 0x4A → notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#QuestErrorLessThanSixMembers
type QuestErrorLessThanSixMembers struct{ mode byte }

func NewQuestErrorLessThanSixMembers(mode byte) QuestErrorLessThanSixMembers {
	return QuestErrorLessThanSixMembers{mode: mode}
}
func (m QuestErrorLessThanSixMembers) Operation() string { return GuildOperationWriter }
func (m QuestErrorLessThanSixMembers) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m QuestErrorLessThanSixMembers) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *QuestErrorLessThanSixMembers) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// QuestErrorDisconnected — THE_USER_THAT_REGISTERED_HAS_DISCONNECTED
// (case 0x4B). Mode-only (v83 OnGuildResult@0xa37490, case 0x4B → notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#QuestErrorDisconnected
type QuestErrorDisconnected struct{ mode byte }

func NewQuestErrorDisconnected(mode byte) QuestErrorDisconnected {
	return QuestErrorDisconnected{mode: mode}
}
func (m QuestErrorDisconnected) Operation() string { return GuildOperationWriter }
func (m QuestErrorDisconnected) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m QuestErrorDisconnected) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *QuestErrorDisconnected) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// --- Target-bearing invite-error arms (discrete per mode; {mode,target}) ------
//
// These three arms each read a trailing player-name string from the wire and
// %s-substitute it into a StringPool message. Verified case-by-case in the v83
// OnGuildResult switch (sub @0xa37490): cases 0x35/0x36/0x37 each begin with
// CInPacket::DecodeStr(targetName) (SP_334 / SP_2723 / SP_335). Contradicts the
// "F4 mode-only" note in context.md §10 — IDA shows a wire string read, so these
// remain {mode,target} (the retired ErrorMessageWithTarget split into 3 discrete
// structs). v83/v84/v87/jms byte-identical shape; v95 mode bytes shifted (yaml).

// InviteErrorNotAcceptingInvites — IS_CURRENTLY_NOT_ACCEPTING (case 0x35).
// Decode1(mode) + DecodeStr(target). v83 OnGuildResult@0xa37490 case 0x35 L273.
// packet-audit:fname CWvsContext::OnGuildResult#InviteErrorNotAcceptingInvites
type InviteErrorNotAcceptingInvites struct {
	mode   byte
	target string
}

func NewInviteErrorNotAcceptingInvites(mode byte, target string) InviteErrorNotAcceptingInvites {
	return InviteErrorNotAcceptingInvites{mode: mode, target: target}
}
func (m InviteErrorNotAcceptingInvites) Operation() string { return GuildOperationWriter }
func (m InviteErrorNotAcceptingInvites) String() string {
	return fmt.Sprintf("mode [%d], target [%s]", m.mode, m.target)
}
func (m InviteErrorNotAcceptingInvites) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte {
		w.WriteByte(m.mode)          // dispatcher mode byte
		w.WriteAsciiString(m.target) // DecodeStr target name (case 0x35 L273)
		return w.Bytes()
	}
}
func (m *InviteErrorNotAcceptingInvites) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		m.mode = r.ReadByte()
		m.target = r.ReadAsciiString()
	}
}

// InviteErrorAnotherInvite — IS_TAKING_CARE_OF_ANOTHER_INVITATION
// (case 0x36). Decode1(mode) + DecodeStr(target). v83 case 0x36 L289.
// packet-audit:fname CWvsContext::OnGuildResult#InviteErrorAnotherInvite
type InviteErrorAnotherInvite struct {
	mode   byte
	target string
}

func NewInviteErrorAnotherInvite(mode byte, target string) InviteErrorAnotherInvite {
	return InviteErrorAnotherInvite{mode: mode, target: target}
}
func (m InviteErrorAnotherInvite) Operation() string { return GuildOperationWriter }
func (m InviteErrorAnotherInvite) String() string {
	return fmt.Sprintf("mode [%d], target [%s]", m.mode, m.target)
}
func (m InviteErrorAnotherInvite) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.target) // DecodeStr target name (case 0x36 L289)
		return w.Bytes()
	}
}
func (m *InviteErrorAnotherInvite) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		m.mode = r.ReadByte()
		m.target = r.ReadAsciiString()
	}
}

// InviteDenied — HAS_DENIED_YOUR_GUILD_INVITATION (case 0x37).
// Decode1(mode) + DecodeStr(target). v83 case 0x37 L305.
// packet-audit:fname CWvsContext::OnGuildResult#InviteDenied
type InviteDenied struct {
	mode   byte
	target string
}

func NewInviteDenied(mode byte, target string) InviteDenied {
	return InviteDenied{mode: mode, target: target}
}
func (m InviteDenied) Operation() string { return GuildOperationWriter }
func (m InviteDenied) String() string {
	return fmt.Sprintf("mode [%d], target [%s]", m.mode, m.target)
}
func (m InviteDenied) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.target) // DecodeStr target name (case 0x37 L305)
		return w.Bytes()
	}
}
func (m *InviteDenied) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		m.mode = r.ReadByte()
		m.target = r.ReadAsciiString()
	}
}

// EmblemChange

// packet-audit:fname CWvsContext::OnGuildResult#EmblemChange
type EmblemChange struct {
	mode                byte
	guildId             uint32
	logo                uint16
	logoColor           byte
	logoBackground      uint16
	logoBackgroundColor byte
}

func NewEmblemChange(mode byte, guildId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) EmblemChange {
	return EmblemChange{mode: mode, guildId: guildId, logo: logo, logoColor: logoColor, logoBackground: logoBackground, logoBackgroundColor: logoBackgroundColor}
}

func (m EmblemChange) Operation() string { return GuildOperationWriter }
func (m EmblemChange) String() string {
	return fmt.Sprintf("mode [%d], guildId [%d], logo [%d], logoColor [%d], logoBackground [%d], logoBackgroundColor [%d]", m.mode, m.guildId, m.logo, m.logoColor, m.logoBackground, m.logoBackgroundColor)
}

func (m EmblemChange) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.guildId)
		w.WriteShort(m.logoBackground)
		w.WriteByte(m.logoBackgroundColor)
		w.WriteShort(m.logo)
		w.WriteByte(m.logoColor)
		return w.Bytes()
	}
}

func (m *EmblemChange) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.guildId = r.ReadUint32()
		m.logoBackground = r.ReadUint16()
		m.logoBackgroundColor = r.ReadByte()
		m.logo = r.ReadUint16()
		m.logoColor = r.ReadByte()
	}
}

// MemberStatusUpdate

// packet-audit:fname CWvsContext::OnGuildResult#MemberStatusUpdate
type MemberStatusUpdate struct {
	mode        byte
	guildId     uint32
	characterId uint32
	online      bool
}

func NewMemberStatusUpdate(mode byte, guildId uint32, characterId uint32, online bool) MemberStatusUpdate {
	return MemberStatusUpdate{mode: mode, guildId: guildId, characterId: characterId, online: online}
}

func (m MemberStatusUpdate) Operation() string { return GuildOperationWriter }
func (m MemberStatusUpdate) String() string {
	return fmt.Sprintf("mode [%d], guildId [%d], characterId [%d], online [%t]", m.mode, m.guildId, m.characterId, m.online)
}

func (m MemberStatusUpdate) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.guildId)
		w.WriteInt(m.characterId)
		w.WriteBool(m.online)
		return w.Bytes()
	}
}

func (m *MemberStatusUpdate) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.guildId = r.ReadUint32()
		m.characterId = r.ReadUint32()
		m.online = r.ReadBool()
	}
}

// MemberTitleUpdate

// packet-audit:fname CWvsContext::OnGuildResult#MemberTitleUpdate
type MemberTitleUpdate struct {
	mode        byte
	guildId     uint32
	characterId uint32
	title       byte
}

func NewMemberTitleUpdate(mode byte, guildId uint32, characterId uint32, title byte) MemberTitleUpdate {
	return MemberTitleUpdate{mode: mode, guildId: guildId, characterId: characterId, title: title}
}

func (m MemberTitleUpdate) Operation() string { return GuildOperationWriter }
func (m MemberTitleUpdate) String() string {
	return fmt.Sprintf("mode [%d], guildId [%d], characterId [%d], title [%d]", m.mode, m.guildId, m.characterId, m.title)
}

func (m MemberTitleUpdate) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.guildId)
		w.WriteInt(m.characterId)
		w.WriteByte(m.title)
		return w.Bytes()
	}
}

func (m *MemberTitleUpdate) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.guildId = r.ReadUint32()
		m.characterId = r.ReadUint32()
		m.title = r.ReadByte()
	}
}

// NoticeChange

// packet-audit:fname CWvsContext::OnGuildResult#NoticeChange
type NoticeChange struct {
	mode    byte
	guildId uint32
	notice  string
}

func NewNoticeChange(mode byte, guildId uint32, notice string) NoticeChange {
	return NoticeChange{mode: mode, guildId: guildId, notice: notice}
}

func (m NoticeChange) Operation() string { return GuildOperationWriter }
func (m NoticeChange) String() string {
	return fmt.Sprintf("mode [%d], guildId [%d], notice [%s]", m.mode, m.guildId, m.notice)
}

func (m NoticeChange) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.guildId)
		w.WriteAsciiString(m.notice)
		return w.Bytes()
	}
}

func (m *NoticeChange) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.guildId = r.ReadUint32()
		m.notice = r.ReadAsciiString()
	}
}

// MemberLeft

// packet-audit:fname CWvsContext::OnGuildResult#MemberLeft
type MemberLeft struct {
	mode        byte
	guildId     uint32
	characterId uint32
	name        string
}

func NewMemberLeft(mode byte, guildId uint32, characterId uint32, name string) MemberLeft {
	return MemberLeft{mode: mode, guildId: guildId, characterId: characterId, name: name}
}

func (m MemberLeft) Operation() string { return GuildOperationWriter }
func (m MemberLeft) String() string {
	return fmt.Sprintf("mode [%d], guildId [%d], characterId [%d], name [%s]", m.mode, m.guildId, m.characterId, m.name)
}

func (m MemberLeft) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.guildId)
		w.WriteInt(m.characterId)
		w.WriteAsciiString(m.name)
		return w.Bytes()
	}
}

func (m *MemberLeft) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.guildId = r.ReadUint32()
		m.characterId = r.ReadUint32()
		m.name = r.ReadAsciiString()
	}
}

// MemberExpel

// packet-audit:fname CWvsContext::OnGuildResult#MemberExpel
type MemberExpel struct {
	mode        byte
	guildId     uint32
	characterId uint32
	name        string
}

func NewMemberExpel(mode byte, guildId uint32, characterId uint32, name string) MemberExpel {
	return MemberExpel{mode: mode, guildId: guildId, characterId: characterId, name: name}
}

func (m MemberExpel) Operation() string { return GuildOperationWriter }
func (m MemberExpel) String() string {
	return fmt.Sprintf("mode [%d], guildId [%d], characterId [%d], name [%s]", m.mode, m.guildId, m.characterId, m.name)
}

func (m MemberExpel) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.guildId)
		w.WriteInt(m.characterId)
		w.WriteAsciiString(m.name)
		return w.Bytes()
	}
}

func (m *MemberExpel) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.guildId = r.ReadUint32()
		m.characterId = r.ReadUint32()
		m.name = r.ReadAsciiString()
	}
}

// MemberJoined

// packet-audit:fname CWvsContext::OnGuildResult#MemberJoined
type MemberJoined struct {
	mode          byte
	guildId       uint32
	characterId   uint32
	name          string
	jobId         uint16
	level         byte
	title         byte
	online        bool
	allianceTitle byte
}

func NewMemberJoined(mode byte, guildId uint32, characterId uint32, name string, jobId uint16, level byte, title byte, online bool, allianceTitle byte) MemberJoined {
	return MemberJoined{mode: mode, guildId: guildId, characterId: characterId, name: name, jobId: jobId, level: level, title: title, online: online, allianceTitle: allianceTitle}
}

func (m MemberJoined) Operation() string { return GuildOperationWriter }
func (m MemberJoined) String() string {
	return fmt.Sprintf("mode [%d], guildId [%d], characterId [%d], name [%s]", m.mode, m.guildId, m.characterId, m.name)
}

func (m MemberJoined) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.guildId)
		w.WriteInt(m.characterId)
		gm := model.GuildMember{
			Name:          m.name,
			JobId:         m.jobId,
			Level:         m.level,
			Title:         m.title,
			Online:        m.online,
			AllianceTitle: m.allianceTitle,
		}
		w.WriteByteArray(gm.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *MemberJoined) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.guildId = r.ReadUint32()
		m.characterId = r.ReadUint32()
		m.name = model.ReadPaddedString(r, 13)
		m.jobId = uint16(r.ReadUint32())
		m.level = byte(r.ReadUint32())
		m.title = byte(r.ReadUint32())
		var onlineVal uint32
		onlineVal = r.ReadUint32()
		m.online = onlineVal == 1
		_ = r.ReadUint32() // signature
		m.allianceTitle = byte(r.ReadUint32())
	}
}

// Invite

// packet-audit:fname CWvsContext::OnGuildResult#Invite
type Invite struct {
	mode           byte
	guildId        uint32
	originatorName string
	unknown        uint32
	skillId        uint32
}

func NewInvite(mode byte, guildId uint32, originatorName string, unknown uint32, skillId uint32) Invite {
	return Invite{mode: mode, guildId: guildId, originatorName: originatorName, unknown: unknown, skillId: skillId}
}

func (m Invite) Operation() string { return GuildOperationWriter }
func (m Invite) String() string {
	return fmt.Sprintf("mode [%d], guildId [%d], originatorName [%s], unknown [%d], skillId [%d]", m.mode, m.guildId, m.originatorName, m.unknown, m.skillId)
}

func (m Invite) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	// INVITE mode byte is 0x05 in every version; the BODY differs.
	//   v83 OnGuildResult@0xa37490 case 5 (L1316 !v7 → L1319-1320): Decode4(guildId)
	//       + DecodeStr(inviterName) ONLY — no trailing ints.
	//   v84 OnGuildResult@0xa82e2b case 5 (L1209 !v7 → L1212-1216): Decode4(guildId)
	//       + DecodeStr(inviterName) + Decode4(unk) + Decode4(skillId) — the 2 trailing
	//       ints ARE read (live v84 IDB, task-103 F3 re-verified). So v84 follows the
	//       v87+ body, NOT v83 — the OPPOSITE of the usual off-by-one.
	//   v87/v95/jms: same as v84 (guildId+name+unk+skillId).
	// Gate boundary is therefore 84, not 87: GMS >= 84 or JMS read the trailing ints.
	trailingInts := (t.IsRegion("GMS") && t.MajorAtLeast(84)) || t.Region() == "JMS"
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.guildId)
		w.WriteAsciiString(m.originatorName)
		if trailingInts {
			w.WriteInt(m.unknown)
			w.WriteInt(m.skillId)
		}
		return w.Bytes()
	}
}

func (m *Invite) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	// trailingInts gate: see Encode comment above (boundary 84, F3).
	trailingInts := (t.IsRegion("GMS") && t.MajorAtLeast(84)) || t.Region() == "JMS"
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.guildId = r.ReadUint32()
		m.originatorName = r.ReadAsciiString()
		if trailingInts {
			m.unknown = r.ReadUint32()
			m.skillId = r.ReadUint32()
		}
	}
}

// TitleChange

// packet-audit:fname CWvsContext::OnGuildResult#TitleChange
type TitleChange struct {
	mode    byte
	guildId uint32
	titles  [5]string
}

func NewTitleChange(mode byte, guildId uint32, titles [5]string) TitleChange {
	return TitleChange{mode: mode, guildId: guildId, titles: titles}
}

func (m TitleChange) Operation() string { return GuildOperationWriter }
func (m TitleChange) String() string {
	return fmt.Sprintf("mode [%d], guildId [%d]", m.mode, m.guildId)
}

func (m TitleChange) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.guildId)
		for _, title := range m.titles {
			w.WriteAsciiString(title)
		}
		return w.Bytes()
	}
}

func (m *TitleChange) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.guildId = r.ReadUint32()
		for i := 0; i < 5; i++ {
			m.titles[i] = r.ReadAsciiString()
		}
	}
}

// Disband

// packet-audit:fname CWvsContext::OnGuildResult#Disband
type Disband struct {
	mode    byte
	guildId uint32
}

func NewDisband(mode byte, guildId uint32) Disband {
	return Disband{mode: mode, guildId: guildId}
}

func (m Disband) Operation() string { return GuildOperationWriter }
func (m Disband) String() string {
	return fmt.Sprintf("mode [%d], guildId [%d]", m.mode, m.guildId)
}

func (m Disband) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.guildId)
		return w.Bytes()
	}
}

func (m *Disband) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.guildId = r.ReadUint32()
	}
}

// CapacityChange

// packet-audit:fname CWvsContext::OnGuildResult#CapacityChange
type CapacityChange struct {
	mode     byte
	guildId  uint32
	capacity byte
}

func NewCapacityChange(mode byte, guildId uint32, capacity byte) CapacityChange {
	return CapacityChange{mode: mode, guildId: guildId, capacity: capacity}
}

func (m CapacityChange) Operation() string { return GuildOperationWriter }
func (m CapacityChange) String() string {
	return fmt.Sprintf("mode [%d], guildId [%d], capacity [%d]", m.mode, m.guildId, m.capacity)
}

func (m CapacityChange) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.guildId)
		w.WriteByte(m.capacity)
		return w.Bytes()
	}
}

func (m *CapacityChange) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.guildId = r.ReadUint32()
		m.capacity = r.ReadByte()
	}
}

// --- Structured arms previously without a discrete struct (task-103) ----------
//
// MEMBER_UPDATE / SHOW_TITLES / QUEST_WAITING_NOTICE / BOARD_AUTH_KEY_UPDATE /
// SET_SKILL_RESPONSE — read orders verified case-by-case in v83 OnGuildResult
// (sub @0xa37490). v83/v84/v87/jms byte-identical; v95 mode bytes shifted (yaml).

// MemberUpdate — MEMBER_UPDATE (case 0x3C). After the guildId match check,
// reads Decode4(charId) + Decode4(level) + Decode4(job). v83 OnGuildResult@0xa37490
// case 0x3C (L379 guildId-check, L381 charId, L382 level, L383 job).
// packet-audit:fname CWvsContext::OnGuildResult#MemberUpdate
type MemberUpdate struct {
	mode        byte
	guildId     uint32
	characterId uint32
	level       uint32
	job         uint32
}

func NewMemberUpdate(mode byte, guildId uint32, characterId uint32, level uint32, job uint32) MemberUpdate {
	return MemberUpdate{mode: mode, guildId: guildId, characterId: characterId, level: level, job: job}
}
func (m MemberUpdate) Operation() string { return GuildOperationWriter }
func (m MemberUpdate) String() string {
	return fmt.Sprintf("mode [%d], guildId [%d], characterId [%d], level [%d], job [%d]", m.mode, m.guildId, m.characterId, m.level, m.job)
}
func (m MemberUpdate) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.guildId)     // Decode4 guildId (match check)
		w.WriteInt(m.characterId) // Decode4 charId
		w.WriteInt(m.level)       // Decode4 level
		w.WriteInt(m.job)         // Decode4 job
		return w.Bytes()
	}
}
func (m *MemberUpdate) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		m.mode = r.ReadByte()
		m.guildId = r.ReadUint32()
		m.characterId = r.ReadUint32()
		m.level = r.ReadUint32()
		m.job = r.ReadUint32()
	}
}

// GuildTitleEntry is one (name, ints[5]) row of the SHOW_TITLES list.
type GuildTitleEntry struct {
	Name   string
	Values [5]uint32
}

// ShowTitles — SHOW_TITLES (case 0x49). Reads Decode4(guildId) [discarded] +
// Decode4(count) + count×[DecodeStr(name) + 5×Decode4]. v83 OnGuildResult@0xa37490
// case 0x49 (L645 guildId, L646 count, loop L656 name + L662-666 5 ints).
// packet-audit:fname CWvsContext::OnGuildResult#ShowTitles
type ShowTitles struct {
	mode    byte
	guildId uint32
	entries []GuildTitleEntry
}

func NewShowTitles(mode byte, guildId uint32, entries []GuildTitleEntry) ShowTitles {
	return ShowTitles{mode: mode, guildId: guildId, entries: entries}
}
func (m ShowTitles) Operation() string { return GuildOperationWriter }
func (m ShowTitles) String() string {
	return fmt.Sprintf("mode [%d], guildId [%d], entries [%d]", m.mode, m.guildId, len(m.entries))
}
func (m ShowTitles) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.guildId)              // Decode4 guildId (discarded by client)
		w.WriteInt(uint32(len(m.entries))) // Decode4 count (loop bound)
		for _, e := range m.entries {
			w.WriteAsciiString(e.Name) // DecodeStr name
			for _, v := range e.Values {
				w.WriteInt(v) // 5×Decode4
			}
		}
		return w.Bytes()
	}
}
func (m *ShowTitles) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		m.mode = r.ReadByte()
		m.guildId = r.ReadUint32()
		count := r.ReadUint32()
		m.entries = make([]GuildTitleEntry, 0, count)
		for i := uint32(0); i < count; i++ {
			var e GuildTitleEntry
			e.Name = r.ReadAsciiString()
			for j := 0; j < 5; j++ {
				e.Values[j] = r.ReadUint32()
			}
			m.entries = append(m.entries, e)
		}
	}
}

// QuestWaitingNotice — QUEST_WAITING_NOTICE (case 0x4C). Reads Decode1(channel)
// + Decode4(state). v83 OnGuildResult@0xa37490 case 0x4C (L713 channel, L714 state).
// packet-audit:fname CWvsContext::OnGuildResult#QuestWaitingNotice
type QuestWaitingNotice struct {
	mode    byte
	channel byte
	state   uint32
}

func NewQuestWaitingNotice(mode byte, channel byte, state uint32) QuestWaitingNotice {
	return QuestWaitingNotice{mode: mode, channel: channel, state: state}
}
func (m QuestWaitingNotice) Operation() string { return GuildOperationWriter }
func (m QuestWaitingNotice) String() string {
	return fmt.Sprintf("mode [%d], channel [%d], state [%d]", m.mode, m.channel, m.state)
}
func (m QuestWaitingNotice) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.channel) // Decode1 channel
		w.WriteInt(m.state)    // Decode4 state
		return w.Bytes()
	}
}
func (m *QuestWaitingNotice) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		m.mode = r.ReadByte()
		m.channel = r.ReadByte()
		m.state = r.ReadUint32()
	}
}

// BoardAuthKeyUpdate — BOARD_AUTH_KEY_UPDATE (case 0x4D). Reads
// DecodeStr(authKey). v83 OnGuildResult@0xa37490 case 0x4D (L791).
// packet-audit:fname CWvsContext::OnGuildResult#BoardAuthKeyUpdate
type BoardAuthKeyUpdate struct {
	mode    byte
	authKey string
}

func NewBoardAuthKeyUpdate(mode byte, authKey string) BoardAuthKeyUpdate {
	return BoardAuthKeyUpdate{mode: mode, authKey: authKey}
}
func (m BoardAuthKeyUpdate) Operation() string { return GuildOperationWriter }
func (m BoardAuthKeyUpdate) String() string {
	return fmt.Sprintf("mode [%d], authKey [%s]", m.mode, m.authKey)
}
func (m BoardAuthKeyUpdate) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.authKey) // DecodeStr authKey
		return w.Bytes()
	}
}
func (m *BoardAuthKeyUpdate) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		m.mode = r.ReadByte()
		m.authKey = r.ReadAsciiString()
	}
}

// SetSkillResponse — SET_SKILL_RESPONSE (case 0x4E; GMS only, jms-absent).
// Reads Decode1(success); when success!=0 then DecodeStr(message). v83
// OnGuildResult@0xa37490 case 0x4E (L802 success, L804 message in the truthy branch).
// packet-audit:fname CWvsContext::OnGuildResult#SetSkillResponse
type SetSkillResponse struct {
	mode    byte
	success bool
	message string
}

func NewSetSkillResponse(mode byte, success bool, message string) SetSkillResponse {
	return SetSkillResponse{mode: mode, success: success, message: message}
}
func (m SetSkillResponse) Operation() string { return GuildOperationWriter }
func (m SetSkillResponse) String() string {
	return fmt.Sprintf("mode [%d], success [%t], message [%s]", m.mode, m.success, m.message)
}
func (m SetSkillResponse) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteBool(m.success) // Decode1 success flag
		if m.success {
			w.WriteAsciiString(m.message) // DecodeStr message (truthy branch only)
		}
		return w.Bytes()
	}
}
func (m *SetSkillResponse) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		m.mode = r.ReadByte()
		m.success = r.ReadBool()
		if m.success {
			m.message = r.ReadAsciiString()
		}
	}
}
