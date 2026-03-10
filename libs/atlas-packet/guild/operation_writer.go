package guild

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas-packet/model"
)

const GuildOperationWriter = "GuildOperation"

// RequestAgreement

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

// ErrorMessage

type ErrorMessage struct {
	mode byte
}

func NewErrorMessage(mode byte) ErrorMessage {
	return ErrorMessage{mode: mode}
}

func (m ErrorMessage) Operation() string { return GuildOperationWriter }
func (m ErrorMessage) String() string {
	return fmt.Sprintf("mode [%d]", m.mode)
}

func (m ErrorMessage) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ErrorMessage) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// ErrorMessageWithTarget

type ErrorMessageWithTarget struct {
	mode   byte
	target string
}

func NewErrorMessageWithTarget(mode byte, target string) ErrorMessageWithTarget {
	return ErrorMessageWithTarget{mode: mode, target: target}
}

func (m ErrorMessageWithTarget) Operation() string { return GuildOperationWriter }
func (m ErrorMessageWithTarget) String() string {
	return fmt.Sprintf("mode [%d], target [%s]", m.mode, m.target)
}

func (m ErrorMessageWithTarget) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.target)
		return w.Bytes()
	}
}

func (m *ErrorMessageWithTarget) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.target = r.ReadAsciiString()
	}
}

// EmblemChange

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

// InviteW

type InviteW struct {
	mode            byte
	guildId         uint32
	originatorName  string
}

func NewInviteW(mode byte, guildId uint32, originatorName string) InviteW {
	return InviteW{mode: mode, guildId: guildId, originatorName: originatorName}
}

func (m InviteW) Operation() string { return GuildOperationWriter }
func (m InviteW) String() string {
	return fmt.Sprintf("mode [%d], guildId [%d], originatorName [%s]", m.mode, m.guildId, m.originatorName)
}

func (m InviteW) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.guildId)
		w.WriteAsciiString(m.originatorName)
		return w.Bytes()
	}
}

func (m *InviteW) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.guildId = r.ReadUint32()
		m.originatorName = r.ReadAsciiString()
	}
}

// TitleChange

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

type CapacityChange struct {
	mode     byte
	guildId  uint32
	capacity uint32
}

func NewCapacityChange(mode byte, guildId uint32, capacity uint32) CapacityChange {
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
		w.WriteInt(m.capacity)
		return w.Bytes()
	}
}

func (m *CapacityChange) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.guildId = r.ReadUint32()
		m.capacity = r.ReadUint32()
	}
}
