package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

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

func (m ServerListEntry) WorldId() world.Id                  { return m.worldId }
func (m ServerListEntry) WorldName() string                  { return m.worldName }
func (m ServerListEntry) State() byte                        { return m.state }
func (m ServerListEntry) EventMessage() string               { return m.eventMessage }
func (m ServerListEntry) ChannelLoads() []model.ChannelLoad  { return m.channelLoads }
func (m ServerListEntry) Balloons() []model.WorldBalloon     { return m.balloons }
func (m ServerListEntry) Operation() string                  { return ServerListEntryWriter }
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
			w.WriteAsciiString(fmt.Sprintf("%s - %d", m.worldName, x.ChannelId()))
			w.WriteInt(x.Capacity())
			w.WriteByte(byte(m.worldId))
			w.WriteByte(byte(x.ChannelId() - 1))
			w.WriteBool(false) // adult channel
		}

		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
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

		if (t.Region() == "GMS" && t.MajorVersion() > 12) || t.Region() == "JMS" {
			balloonCount := r.ReadUint16()
			m.balloons = make([]model.WorldBalloon, balloonCount)
			for i := uint16(0); i < balloonCount; i++ {
				m.balloons[i].Read(r)
			}
		}
	}
}
