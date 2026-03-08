package login

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
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
		if t.Region() == "GMS" && t.MajorVersion() > 28 {
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

func (m WorldCharacterListRequest) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if t.Region() == "GMS" && t.MajorVersion() > 28 {
			// GMS v28 is not definite here, but this is not present in 28.
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
