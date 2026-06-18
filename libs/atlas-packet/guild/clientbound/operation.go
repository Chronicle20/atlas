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

// RequestAgreement

// packet-audit:fname CWvsContext::OnGuildResult#AgreementResponse
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

// GuildRequestName — REQUEST_NAME (case 0x01). Prompts the create-guild name
// dialog; the sub-handler reads no body. v83 OnGuildResult@0xa37490 if-chain
// (v4==1 → CField::InputGuildName, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#GuildRequestName
type GuildRequestName struct{ mode byte }

func NewGuildRequestName(mode byte) GuildRequestName { return GuildRequestName{mode: mode} }
func (m GuildRequestName) Operation() string         { return GuildOperationWriter }
func (m GuildRequestName) String() string            { return fmt.Sprintf("mode [%d]", m.mode) }
func (m GuildRequestName) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *GuildRequestName) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// GuildRequestEmblem — REQUEST_EMBLEM (case 0x11). Prompts the set-emblem
// dialog; no body. v83 OnGuildResult@0xa37490 (v4==0x11 → CSetGuildMarkDlg, no
// Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#GuildRequestEmblem
type GuildRequestEmblem struct{ mode byte }

func NewGuildRequestEmblem(mode byte) GuildRequestEmblem { return GuildRequestEmblem{mode: mode} }
func (m GuildRequestEmblem) Operation() string           { return GuildOperationWriter }
func (m GuildRequestEmblem) String() string              { return fmt.Sprintf("mode [%d]", m.mode) }
func (m GuildRequestEmblem) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *GuildRequestEmblem) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// GuildCreateErrorNameInUse — THE_NAME_IS_ALREADY_IN_USE (case 0x1C). Mode-only
// (v83 OnGuildResult@0xa37490, v4==0x1C → StringPool notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#GuildCreateErrorNameInUse
type GuildCreateErrorNameInUse struct{ mode byte }

func NewGuildCreateErrorNameInUse(mode byte) GuildCreateErrorNameInUse {
	return GuildCreateErrorNameInUse{mode: mode}
}
func (m GuildCreateErrorNameInUse) Operation() string { return GuildOperationWriter }
func (m GuildCreateErrorNameInUse) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m GuildCreateErrorNameInUse) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *GuildCreateErrorNameInUse) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// GuildCreateErrorDisagreed — SOMEBODY_HAS_DISAGREED (case 0x24). Mode-only
// (v83 OnGuildResult@0xa37490, case 0x24 → notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#GuildCreateErrorDisagreed
type GuildCreateErrorDisagreed struct{ mode byte }

func NewGuildCreateErrorDisagreed(mode byte) GuildCreateErrorDisagreed {
	return GuildCreateErrorDisagreed{mode: mode}
}
func (m GuildCreateErrorDisagreed) Operation() string { return GuildOperationWriter }
func (m GuildCreateErrorDisagreed) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m GuildCreateErrorDisagreed) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *GuildCreateErrorDisagreed) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// GuildCreateError — THE_PROBLEM…FORMING_THE_GUILD (case 0x26). Mode-only
// (v83 OnGuildResult@0xa37490, case 0x26 → notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#GuildCreateError
type GuildCreateError struct{ mode byte }

func NewGuildCreateError(mode byte) GuildCreateError { return GuildCreateError{mode: mode} }
func (m GuildCreateError) Operation() string         { return GuildOperationWriter }
func (m GuildCreateError) String() string            { return fmt.Sprintf("mode [%d]", m.mode) }
func (m GuildCreateError) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *GuildCreateError) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// GuildJoinErrorAlreadyJoined — ALREADY_JOINED_THE_GUILD (case 0x28). Mode-only
// (v83 OnGuildResult@0xa37490, case 0x28 → notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#GuildJoinErrorAlreadyJoined
type GuildJoinErrorAlreadyJoined struct{ mode byte }

func NewGuildJoinErrorAlreadyJoined(mode byte) GuildJoinErrorAlreadyJoined {
	return GuildJoinErrorAlreadyJoined{mode: mode}
}
func (m GuildJoinErrorAlreadyJoined) Operation() string { return GuildOperationWriter }
func (m GuildJoinErrorAlreadyJoined) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m GuildJoinErrorAlreadyJoined) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *GuildJoinErrorAlreadyJoined) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// GuildJoinErrorMaxMembers — MAX_NUMBER_OF_USERS (case 0x29). Mode-only
// (v83 OnGuildResult@0xa37490, case 0x29 → notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#GuildJoinErrorMaxMembers
type GuildJoinErrorMaxMembers struct{ mode byte }

func NewGuildJoinErrorMaxMembers(mode byte) GuildJoinErrorMaxMembers {
	return GuildJoinErrorMaxMembers{mode: mode}
}
func (m GuildJoinErrorMaxMembers) Operation() string { return GuildOperationWriter }
func (m GuildJoinErrorMaxMembers) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m GuildJoinErrorMaxMembers) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *GuildJoinErrorMaxMembers) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// GuildJoinErrorNotInChannel — CHARACTER_CANNOT_BE_FOUND_IN_THE_CURRENT_CHANNEL
// (case 0x2A). Mode-only (v83 OnGuildResult@0xa37490, case 0x2A → notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#GuildJoinErrorNotInChannel
type GuildJoinErrorNotInChannel struct{ mode byte }

func NewGuildJoinErrorNotInChannel(mode byte) GuildJoinErrorNotInChannel {
	return GuildJoinErrorNotInChannel{mode: mode}
}
func (m GuildJoinErrorNotInChannel) Operation() string { return GuildOperationWriter }
func (m GuildJoinErrorNotInChannel) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m GuildJoinErrorNotInChannel) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *GuildJoinErrorNotInChannel) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// GuildMemberQuitErrorNotInGuild — MEMBER_QUIT_ERROR_NOT_IN_GUILD (case 0x2D).
// Mode-only (v83 OnGuildResult@0xa37490, case 0x2D → notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#GuildMemberQuitErrorNotInGuild
type GuildMemberQuitErrorNotInGuild struct{ mode byte }

func NewGuildMemberQuitErrorNotInGuild(mode byte) GuildMemberQuitErrorNotInGuild {
	return GuildMemberQuitErrorNotInGuild{mode: mode}
}
func (m GuildMemberQuitErrorNotInGuild) Operation() string { return GuildOperationWriter }
func (m GuildMemberQuitErrorNotInGuild) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m GuildMemberQuitErrorNotInGuild) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *GuildMemberQuitErrorNotInGuild) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// GuildMemberExpelledErrorNotInGuild — MEMBER_EXPELLED_ERROR_NOT_IN_GUILD
// (case 0x30). Mode-only (v83 OnGuildResult@0xa37490, case 0x30 → notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#GuildMemberExpelledErrorNotInGuild
type GuildMemberExpelledErrorNotInGuild struct{ mode byte }

func NewGuildMemberExpelledErrorNotInGuild(mode byte) GuildMemberExpelledErrorNotInGuild {
	return GuildMemberExpelledErrorNotInGuild{mode: mode}
}
func (m GuildMemberExpelledErrorNotInGuild) Operation() string { return GuildOperationWriter }
func (m GuildMemberExpelledErrorNotInGuild) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m GuildMemberExpelledErrorNotInGuild) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *GuildMemberExpelledErrorNotInGuild) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// GuildDisbandError — THE_PROBLEM…DISBANDING_THE_GUILD (case 0x34). Mode-only
// (v83 OnGuildResult@0xa37490, case 0x34 → GuildNPCSay, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#GuildDisbandError
type GuildDisbandError struct{ mode byte }

func NewGuildDisbandError(mode byte) GuildDisbandError { return GuildDisbandError{mode: mode} }
func (m GuildDisbandError) Operation() string          { return GuildOperationWriter }
func (m GuildDisbandError) String() string             { return fmt.Sprintf("mode [%d]", m.mode) }
func (m GuildDisbandError) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *GuildDisbandError) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// GuildCreateErrorCannotAsAdmin — ADMIN_CANNOT_MAKE_A_GUILD (case 0x38).
// Mode-only (v83 OnGuildResult@0xa37490, case 0x38 → CHATLOG, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#GuildCreateErrorCannotAsAdmin
type GuildCreateErrorCannotAsAdmin struct{ mode byte }

func NewGuildCreateErrorCannotAsAdmin(mode byte) GuildCreateErrorCannotAsAdmin {
	return GuildCreateErrorCannotAsAdmin{mode: mode}
}
func (m GuildCreateErrorCannotAsAdmin) Operation() string { return GuildOperationWriter }
func (m GuildCreateErrorCannotAsAdmin) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m GuildCreateErrorCannotAsAdmin) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *GuildCreateErrorCannotAsAdmin) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// GuildIncreaseCapacityError — THE_PROBLEM…INCREASING_THE_GUILD (case 0x3B).
// Mode-only (v83 OnGuildResult@0xa37490, case 0x3B → notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#GuildIncreaseCapacityError
type GuildIncreaseCapacityError struct{ mode byte }

func NewGuildIncreaseCapacityError(mode byte) GuildIncreaseCapacityError {
	return GuildIncreaseCapacityError{mode: mode}
}
func (m GuildIncreaseCapacityError) Operation() string { return GuildOperationWriter }
func (m GuildIncreaseCapacityError) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m GuildIncreaseCapacityError) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *GuildIncreaseCapacityError) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// GuildQuestErrorLessThanSixMembers — THERE_ARE_LESS_THAN_6_MEMBERS (case 0x4A).
// Mode-only (v83 OnGuildResult@0xa37490, case 0x4A → notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#GuildQuestErrorLessThanSixMembers
type GuildQuestErrorLessThanSixMembers struct{ mode byte }

func NewGuildQuestErrorLessThanSixMembers(mode byte) GuildQuestErrorLessThanSixMembers {
	return GuildQuestErrorLessThanSixMembers{mode: mode}
}
func (m GuildQuestErrorLessThanSixMembers) Operation() string { return GuildOperationWriter }
func (m GuildQuestErrorLessThanSixMembers) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m GuildQuestErrorLessThanSixMembers) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *GuildQuestErrorLessThanSixMembers) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) { m.mode = r.ReadByte() }
}

// GuildQuestErrorDisconnected — THE_USER_THAT_REGISTERED_HAS_DISCONNECTED
// (case 0x4B). Mode-only (v83 OnGuildResult@0xa37490, case 0x4B → notice, no Decode*).
// packet-audit:fname CWvsContext::OnGuildResult#GuildQuestErrorDisconnected
type GuildQuestErrorDisconnected struct{ mode byte }

func NewGuildQuestErrorDisconnected(mode byte) GuildQuestErrorDisconnected {
	return GuildQuestErrorDisconnected{mode: mode}
}
func (m GuildQuestErrorDisconnected) Operation() string { return GuildOperationWriter }
func (m GuildQuestErrorDisconnected) String() string    { return fmt.Sprintf("mode [%d]", m.mode) }
func (m GuildQuestErrorDisconnected) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte { w.WriteByte(m.mode); return w.Bytes() }
}
func (m *GuildQuestErrorDisconnected) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
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

// GuildInviteErrorNotAcceptingInvites — IS_CURRENTLY_NOT_ACCEPTING (case 0x35).
// Decode1(mode) + DecodeStr(target). v83 OnGuildResult@0xa37490 case 0x35 L273.
// packet-audit:fname CWvsContext::OnGuildResult#GuildInviteErrorNotAcceptingInvites
type GuildInviteErrorNotAcceptingInvites struct {
	mode   byte
	target string
}

func NewGuildInviteErrorNotAcceptingInvites(mode byte, target string) GuildInviteErrorNotAcceptingInvites {
	return GuildInviteErrorNotAcceptingInvites{mode: mode, target: target}
}
func (m GuildInviteErrorNotAcceptingInvites) Operation() string { return GuildOperationWriter }
func (m GuildInviteErrorNotAcceptingInvites) String() string {
	return fmt.Sprintf("mode [%d], target [%s]", m.mode, m.target)
}
func (m GuildInviteErrorNotAcceptingInvites) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte {
		w.WriteByte(m.mode)            // dispatcher mode byte
		w.WriteAsciiString(m.target)   // DecodeStr target name (case 0x35 L273)
		return w.Bytes()
	}
}
func (m *GuildInviteErrorNotAcceptingInvites) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		m.mode = r.ReadByte()
		m.target = r.ReadAsciiString()
	}
}

// GuildInviteErrorAnotherInvite — IS_TAKING_CARE_OF_ANOTHER_INVITATION
// (case 0x36). Decode1(mode) + DecodeStr(target). v83 case 0x36 L289.
// packet-audit:fname CWvsContext::OnGuildResult#GuildInviteErrorAnotherInvite
type GuildInviteErrorAnotherInvite struct {
	mode   byte
	target string
}

func NewGuildInviteErrorAnotherInvite(mode byte, target string) GuildInviteErrorAnotherInvite {
	return GuildInviteErrorAnotherInvite{mode: mode, target: target}
}
func (m GuildInviteErrorAnotherInvite) Operation() string { return GuildOperationWriter }
func (m GuildInviteErrorAnotherInvite) String() string {
	return fmt.Sprintf("mode [%d], target [%s]", m.mode, m.target)
}
func (m GuildInviteErrorAnotherInvite) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.target) // DecodeStr target name (case 0x36 L289)
		return w.Bytes()
	}
}
func (m *GuildInviteErrorAnotherInvite) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		m.mode = r.ReadByte()
		m.target = r.ReadAsciiString()
	}
}

// GuildInviteDenied — HAS_DENIED_YOUR_GUILD_INVITATION (case 0x37).
// Decode1(mode) + DecodeStr(target). v83 case 0x37 L305.
// packet-audit:fname CWvsContext::OnGuildResult#GuildInviteDenied
type GuildInviteDenied struct {
	mode   byte
	target string
}

func NewGuildInviteDenied(mode byte, target string) GuildInviteDenied {
	return GuildInviteDenied{mode: mode, target: target}
}
func (m GuildInviteDenied) Operation() string { return GuildOperationWriter }
func (m GuildInviteDenied) String() string {
	return fmt.Sprintf("mode [%d], target [%s]", m.mode, m.target)
}
func (m GuildInviteDenied) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.target) // DecodeStr target name (case 0x37 L305)
		return w.Bytes()
	}
}
func (m *GuildInviteDenied) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
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

// GuildMemberUpdate — MEMBER_UPDATE (case 0x3C). After the guildId match check,
// reads Decode4(charId) + Decode4(level) + Decode4(job). v83 OnGuildResult@0xa37490
// case 0x3C (L379 guildId-check, L381 charId, L382 level, L383 job).
// packet-audit:fname CWvsContext::OnGuildResult#GuildMemberUpdate
type GuildMemberUpdate struct {
	mode        byte
	guildId     uint32
	characterId uint32
	level       uint32
	job         uint32
}

func NewGuildMemberUpdate(mode byte, guildId uint32, characterId uint32, level uint32, job uint32) GuildMemberUpdate {
	return GuildMemberUpdate{mode: mode, guildId: guildId, characterId: characterId, level: level, job: job}
}
func (m GuildMemberUpdate) Operation() string { return GuildOperationWriter }
func (m GuildMemberUpdate) String() string {
	return fmt.Sprintf("mode [%d], guildId [%d], characterId [%d], level [%d], job [%d]", m.mode, m.guildId, m.characterId, m.level, m.job)
}
func (m GuildMemberUpdate) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
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
func (m *GuildMemberUpdate) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
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

// GuildShowTitles — SHOW_TITLES (case 0x49). Reads Decode4(guildId) [discarded] +
// Decode4(count) + count×[DecodeStr(name) + 5×Decode4]. v83 OnGuildResult@0xa37490
// case 0x49 (L645 guildId, L646 count, loop L656 name + L662-666 5 ints).
// packet-audit:fname CWvsContext::OnGuildResult#GuildShowTitles
type GuildShowTitles struct {
	mode    byte
	guildId uint32
	entries []GuildTitleEntry
}

func NewGuildShowTitles(mode byte, guildId uint32, entries []GuildTitleEntry) GuildShowTitles {
	return GuildShowTitles{mode: mode, guildId: guildId, entries: entries}
}
func (m GuildShowTitles) Operation() string { return GuildOperationWriter }
func (m GuildShowTitles) String() string {
	return fmt.Sprintf("mode [%d], guildId [%d], entries [%d]", m.mode, m.guildId, len(m.entries))
}
func (m GuildShowTitles) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.guildId)                // Decode4 guildId (discarded by client)
		w.WriteInt(uint32(len(m.entries)))   // Decode4 count (loop bound)
		for _, e := range m.entries {
			w.WriteAsciiString(e.Name) // DecodeStr name
			for _, v := range e.Values {
				w.WriteInt(v) // 5×Decode4
			}
		}
		return w.Bytes()
	}
}
func (m *GuildShowTitles) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
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

// GuildQuestWaitingNotice — QUEST_WAITING_NOTICE (case 0x4C). Reads Decode1(channel)
// + Decode4(state). v83 OnGuildResult@0xa37490 case 0x4C (L713 channel, L714 state).
// packet-audit:fname CWvsContext::OnGuildResult#GuildQuestWaitingNotice
type GuildQuestWaitingNotice struct {
	mode    byte
	channel byte
	state   uint32
}

func NewGuildQuestWaitingNotice(mode byte, channel byte, state uint32) GuildQuestWaitingNotice {
	return GuildQuestWaitingNotice{mode: mode, channel: channel, state: state}
}
func (m GuildQuestWaitingNotice) Operation() string { return GuildOperationWriter }
func (m GuildQuestWaitingNotice) String() string {
	return fmt.Sprintf("mode [%d], channel [%d], state [%d]", m.mode, m.channel, m.state)
}
func (m GuildQuestWaitingNotice) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.channel) // Decode1 channel
		w.WriteInt(m.state)    // Decode4 state
		return w.Bytes()
	}
}
func (m *GuildQuestWaitingNotice) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		m.mode = r.ReadByte()
		m.channel = r.ReadByte()
		m.state = r.ReadUint32()
	}
}

// GuildBoardAuthKeyUpdate — BOARD_AUTH_KEY_UPDATE (case 0x4D). Reads
// DecodeStr(authKey). v83 OnGuildResult@0xa37490 case 0x4D (L791).
// packet-audit:fname CWvsContext::OnGuildResult#GuildBoardAuthKeyUpdate
type GuildBoardAuthKeyUpdate struct {
	mode    byte
	authKey string
}

func NewGuildBoardAuthKeyUpdate(mode byte, authKey string) GuildBoardAuthKeyUpdate {
	return GuildBoardAuthKeyUpdate{mode: mode, authKey: authKey}
}
func (m GuildBoardAuthKeyUpdate) Operation() string { return GuildOperationWriter }
func (m GuildBoardAuthKeyUpdate) String() string {
	return fmt.Sprintf("mode [%d], authKey [%s]", m.mode, m.authKey)
}
func (m GuildBoardAuthKeyUpdate) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.authKey) // DecodeStr authKey
		return w.Bytes()
	}
}
func (m *GuildBoardAuthKeyUpdate) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		m.mode = r.ReadByte()
		m.authKey = r.ReadAsciiString()
	}
}

// GuildSetSkillResponse — SET_SKILL_RESPONSE (case 0x4E; GMS only, jms-absent).
// Reads Decode1(success); when success!=0 then DecodeStr(message). v83
// OnGuildResult@0xa37490 case 0x4E (L802 success, L804 message in the truthy branch).
// packet-audit:fname CWvsContext::OnGuildResult#GuildSetSkillResponse
type GuildSetSkillResponse struct {
	mode    byte
	success bool
	message string
}

func NewGuildSetSkillResponse(mode byte, success bool, message string) GuildSetSkillResponse {
	return GuildSetSkillResponse{mode: mode, success: success, message: message}
}
func (m GuildSetSkillResponse) Operation() string { return GuildOperationWriter }
func (m GuildSetSkillResponse) String() string {
	return fmt.Sprintf("mode [%d], success [%t], message [%s]", m.mode, m.success, m.message)
}
func (m GuildSetSkillResponse) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
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
func (m *GuildSetSkillResponse) Decode(_ logrus.FieldLogger, _ context.Context) func(*request.Reader, map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		m.mode = r.ReadByte()
		m.success = r.ReadBool()
		if m.success {
			m.message = r.ReadAsciiString()
		}
	}
}
