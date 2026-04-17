package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
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
}

func NewServerListEntry(worldId world.Id, worldName string, state byte, eventMessage string, channelLoads []model.ChannelLoad) ServerListEntry {
	return ServerListEntry{
		worldId:      worldId,
		worldName:    worldName,
		state:        state,
		eventMessage: eventMessage,
		channelLoads: channelLoads,
	}
}

func (m ServerListEntry) WorldId() world.Id              { return m.worldId }
func (m ServerListEntry) WorldName() string               { return m.worldName }
func (m ServerListEntry) State() byte                     { return m.state }
func (m ServerListEntry) EventMessage() string            { return m.eventMessage }
func (m ServerListEntry) ChannelLoads() []model.ChannelLoad { return m.channelLoads }
func (m ServerListEntry) Operation() string               { return ServerListEntryWriter }
func (m ServerListEntry) String() string {
	return fmt.Sprintf("worldId [%d], worldName [%s], channels [%d]", m.worldId, m.worldName, len(m.channelLoads))
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
			w.WriteByte(1)
			w.WriteByte(byte(x.ChannelId() - 1))
			w.WriteBool(false) // adult channel
		}

		if t.Region() == "GMS" {
			if t.MajorVersion() > 12 {
				w.WriteShort(0) // balloon size
			}
		} else if t.Region() == "JMS" {
			w.WriteShort(0) // balloon size
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
			_ = r.ReadAsciiString()      // channel name (e.g. "Scania - 1")
			capacity := r.ReadUint32()    // capacity
			_ = r.ReadByte()             // 1
			channelId := r.ReadByte() + 1 // channelId (stored as id-1)
			_ = r.ReadBool()             // adult channel
			m.channelLoads[i] = model.NewChannelLoad(channel.Id(channelId), capacity)
		}

		if t.Region() == "GMS" {
			if t.MajorVersion() > 12 {
				_ = r.ReadUint16() // balloon size
			}
		} else if t.Region() == "JMS" {
			_ = r.ReadUint16() // balloon size
		}
	}
}
