package guild

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas-packet/model"
)

const GuildInfoWriter = "GuildInfo"

type GuildMemberInfo struct {
	CharacterId   uint32
	Name          string
	JobId         uint16
	Level         byte
	Title         byte
	Online        bool
	Signature     uint32
	AllianceTitle byte
}

type Info struct {
	inGuild             bool
	guildId             uint32
	name                string
	titles              [5]string
	members             []GuildMemberInfo
	capacity            uint32
	logoBackground      uint16
	logoBackgroundColor byte
	logo                uint16
	logoColor           byte
	notice              string
	points              uint32
	allianceId          uint32
}

func NewInfo(inGuild bool, guildId uint32, name string, titles [5]string, members []GuildMemberInfo, capacity uint32, logoBackground uint16, logoBackgroundColor byte, logo uint16, logoColor byte, notice string, points uint32, allianceId uint32) Info {
	return Info{
		inGuild:             inGuild,
		guildId:             guildId,
		name:                name,
		titles:              titles,
		members:             members,
		capacity:            capacity,
		logoBackground:      logoBackground,
		logoBackgroundColor: logoBackgroundColor,
		logo:                logo,
		logoColor:           logoColor,
		notice:              notice,
		points:              points,
		allianceId:          allianceId,
	}
}

func (m Info) Operation() string { return GuildInfoWriter }
func (m Info) String() string {
	return fmt.Sprintf("inGuild [%t], guildId [%d], name [%s]", m.inGuild, m.guildId, m.name)
}

func (m Info) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(0x1A)
		w.WriteBool(m.inGuild)
		if !m.inGuild {
			return w.Bytes()
		}
		w.WriteInt(m.guildId)
		w.WriteAsciiString(m.name)
		for _, title := range m.titles {
			w.WriteAsciiString(title)
		}
		w.WriteByte(byte(len(m.members)))
		for _, member := range m.members {
			w.WriteInt(member.CharacterId)
		}
		for _, member := range m.members {
			gm := model.GuildMember{
				Name:          member.Name,
				JobId:         member.JobId,
				Level:         member.Level,
				Title:         member.Title,
				Online:        member.Online,
				Signature:     member.Signature,
				AllianceTitle: member.AllianceTitle,
			}
			w.WriteByteArray(gm.Encode(l, ctx)(options))
		}
		w.WriteInt(m.capacity)
		w.WriteShort(m.logoBackground)
		w.WriteByte(m.logoBackgroundColor)
		w.WriteShort(m.logo)
		w.WriteByte(m.logoColor)
		w.WriteAsciiString(m.notice)
		w.WriteInt(m.points)
		w.WriteInt(m.allianceId)
		return w.Bytes()
	}
}

func (m *Info) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		_ = r.ReadByte() // 0x1A
		m.inGuild = r.ReadBool()
		if !m.inGuild {
			return
		}
		m.guildId = r.ReadUint32()
		m.name = r.ReadAsciiString()
		for i := 0; i < 5; i++ {
			m.titles[i] = r.ReadAsciiString()
		}
		memberCount := r.ReadByte()
		memberIds := make([]uint32, memberCount)
		for i := byte(0); i < memberCount; i++ {
			memberIds[i] = r.ReadUint32()
		}
		m.members = make([]GuildMemberInfo, memberCount)
		for i := byte(0); i < memberCount; i++ {
			m.members[i].CharacterId = memberIds[i]
			m.members[i].Name = model.ReadPaddedString(r, 13)
			m.members[i].JobId = uint16(r.ReadUint32())
			m.members[i].Level = byte(r.ReadUint32())
			m.members[i].Title = byte(r.ReadUint32())
			var onlineVal uint32
			onlineVal = r.ReadUint32()
			m.members[i].Online = onlineVal == 1
			m.members[i].Signature = r.ReadUint32()
			m.members[i].AllianceTitle = byte(r.ReadUint32())
		}
		m.capacity = r.ReadUint32()
		m.logoBackground = r.ReadUint16()
		m.logoBackgroundColor = r.ReadByte()
		m.logo = r.ReadUint16()
		m.logoColor = r.ReadByte()
		m.notice = r.ReadAsciiString()
		m.points = r.ReadUint32()
		m.allianceId = r.ReadUint32()
	}
}
