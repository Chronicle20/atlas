package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// hasWorldBalloons reports whether the world-info body carries the trailing
// balloon block (a count short plus that many {x, y, message} entries).
//
// It arrived between v48 and v61 — the same boundary as the SET_FIELD
// nNotifierCheck short, the NPC-spawn bEnabled byte and the mob team/effectItemId
// pair. IDA: v48 CLogin::OnWorldInformation @0x50120a reads Decode1(worldId)
// @0x501225 and, on the >= 0 arm, DecodeStr(name), Decode1(state) @0x501306,
// DecodeStr(eventMessage), Decode2 @0x50133b, Decode2 @0x501348, Decode1
// @0x501355, Decode1(channelCount) @0x50135d and the per-channel loop
// {DecodeStr, Decode4 @0x5013af, Decode1 @0x5013bc, Decode1 @0x5013c9, Decode1}
// — then returns at 0x5013dc with no further CInPacket call.
//
// v61 @0x56663f reads the identical prefix and then Decode2 @0x5667ea (balloon
// count) followed by a {Decode2, Decode2, DecodeStr} loop @0x56681c/0x566827/
// 0x566830. Writing the count to v48 left two unread bytes on every world entry.
func hasWorldBalloons(t tenant.Model) bool {
	return (t.IsRegion("GMS") && t.MajorAtLeast(61)) || t.Region() == "JMS"
}

const ServerListEntryWriter = "ServerListEntry"

type ServerListEntry struct {
	worldId      world.Id
	worldName    string
	state        byte
	eventMessage string
	channelLoads []model.ChannelLoad
	balloons     []model.WorldBalloon
}

func NewServerListEntry(worldId world.Id, worldName string, state byte, eventMessage string, channelLoads []model.ChannelLoad, balloons []model.WorldBalloon) ServerListEntry {
	return ServerListEntry{
		worldId:      worldId,
		worldName:    worldName,
		state:        state,
		eventMessage: eventMessage,
		channelLoads: channelLoads,
		balloons:     balloons,
	}
}

func (m ServerListEntry) WorldId() world.Id                 { return m.worldId }
func (m ServerListEntry) WorldName() string                 { return m.worldName }
func (m ServerListEntry) State() byte                       { return m.state }
func (m ServerListEntry) EventMessage() string              { return m.eventMessage }
func (m ServerListEntry) ChannelLoads() []model.ChannelLoad { return m.channelLoads }
func (m ServerListEntry) Balloons() []model.WorldBalloon    { return m.balloons }
func (m ServerListEntry) Operation() string                 { return ServerListEntryWriter }
func (m ServerListEntry) String() string {
	return fmt.Sprintf("worldId [%d], worldName [%s], channels [%d], balloons [%d]", m.worldId, m.worldName, len(m.channelLoads), len(m.balloons))
}

func (m ServerListEntry) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(m.worldId))
		w.WriteAsciiString(m.worldName)

		if t.Region() == "GMS" {
			if t.MajorVersion() > 12 {
				w.WriteByte(m.state)
				w.WriteAsciiString(m.eventMessage)
				w.WriteShort(100) // eventExpRate
				w.WriteShort(100) // eventDropRate
				w.WriteByte(0)    // block character creation
			}
		} else if t.Region() == "JMS" {
			w.WriteByte(m.state)
			w.WriteAsciiString(m.eventMessage)
			w.WriteShort(100) // eventExpRate
			w.WriteShort(100) // eventDropRate
		}

		w.WriteByte(byte(len(m.channelLoads)))
		for _, x := range m.channelLoads {
			w.WriteAsciiString(fmt.Sprintf("%s - %d", m.worldName, x.ChannelId()+1))
			w.WriteInt(x.Capacity())
			w.WriteByte(byte(m.worldId))
			w.WriteByte(byte(x.ChannelId() - 1))
			w.WriteBool(false) // adult channel
		}

		if hasWorldBalloons(t) {
			w.WriteShort(uint16(len(m.balloons)))
			for _, b := range m.balloons {
				b.Write(w)
			}
		}

		return w.Bytes()
	}
}

func (m *ServerListEntry) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.worldId = world.Id(r.ReadByte())
		m.worldName = r.ReadAsciiString()

		if t.Region() == "GMS" {
			if t.MajorVersion() > 12 {
				m.state = r.ReadByte()
				m.eventMessage = r.ReadAsciiString()
				_ = r.ReadUint16() // eventExpRate
				_ = r.ReadUint16() // eventDropRate
				_ = r.ReadByte()   // block character creation
			}
		} else if t.Region() == "JMS" {
			m.state = r.ReadByte()
			m.eventMessage = r.ReadAsciiString()
			_ = r.ReadUint16() // eventExpRate
			_ = r.ReadUint16() // eventDropRate
		}

		channelCount := r.ReadByte()
		m.channelLoads = make([]model.ChannelLoad, channelCount)
		for i := byte(0); i < channelCount; i++ {
			_ = r.ReadAsciiString()       // channel name (e.g. "Scania - 1")
			capacity := r.ReadUint32()    // capacity
			_ = r.ReadByte()              // per-channel worldId
			channelId := r.ReadByte() + 1 // channelId (stored as id-1)
			_ = r.ReadBool()              // adult channel
			m.channelLoads[i] = model.NewChannelLoad(channel.Id(channelId), capacity)
		}

		if hasWorldBalloons(t) {
			balloonCount := r.ReadUint16()
			m.balloons = make([]model.WorldBalloon, balloonCount)
			for i := uint16(0); i < balloonCount; i++ {
				m.balloons[i].Read(r)
			}
		}
	}
}
