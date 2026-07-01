package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const WorldCharacterListHandle = "CharacterListWorldHandle"

// WorldCharacterListRequest - CLogin::SendLoginPacket
type WorldCharacterListRequest struct {
	gameStartMode byte
	worldId       world.Id
	channelId     channel.Id
	socketAddr    int32
}

func (m WorldCharacterListRequest) GameStartMode() byte {
	return m.gameStartMode
}

func (m WorldCharacterListRequest) WorldId() world.Id {
	return m.worldId
}

func (m WorldCharacterListRequest) ChannelId() channel.Id {
	return m.channelId
}

func (m WorldCharacterListRequest) SocketAddr() int32 {
	return m.socketAddr
}

func (m WorldCharacterListRequest) Operation() string {
	return WorldCharacterListHandle
}

func (m WorldCharacterListRequest) String() string {
	return fmt.Sprintf("gameStartMode [%d], worldId [%d], channelId [%d], socketAddr [%d]", m.gameStartMode, m.worldId, m.channelId, m.socketAddr)
}

func (m WorldCharacterListRequest) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		// gameStartMode byte absent on the legacy (< v83) char-list request wire.
		// IDA v79 CLogin::SendLoginPacket(sub_5CC905)@0x5cc905 emits
		// COutPacket(5)+Encode1(worldId)+Encode1(channel)+Encode4(ip) only.
		if t.Region() == "GMS" && t.MajorVersion() >= 83 {
			w.WriteByte(m.gameStartMode)
		}
		w.WriteByte(byte(m.worldId))
		w.WriteByte(byte(m.channelId))
		if t.Region() == "GMS" && t.MajorVersion() > 12 {
			w.WriteInt32(m.socketAddr)
		} else if t.Region() == "JMS" {
			w.WriteInt32(m.socketAddr)
		}
		return w.Bytes()
	}
}

func (m *WorldCharacterListRequest) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if t.Region() == "GMS" && t.MajorVersion() >= 83 {
			// gameStartMode absent below v83 (IDA v79 SendLoginPacket@0x5cc905). Mirror of Encode.
			m.gameStartMode = r.ReadByte()
		}
		m.worldId = world.Id(r.ReadByte())
		m.channelId = channel.Id(r.ReadByte())
		if t.Region() == "GMS" && t.MajorVersion() > 12 {
			m.socketAddr = r.ReadInt32()
		} else if t.Region() == "JMS" {
			m.socketAddr = r.ReadInt32()
		}
	}
}
