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
		// socketAddr (client getsockname int) is a v72+ addition. IDA v61
		// CLogin::SendLoginPacket twin sub_564DC9@0x564dc9 emits COutPacket(5)+
		// Encode1(worldId)@0x564efc+Encode1(channelId)@0x564f07 and SendPacket with
		// NO Encode4 — whereas the v72 twin sub_5B1B25@0x5b1b25 adds getsockname->
		// Encode4(socketAddr)@0x5b1c92. Gate the int to GMS>=72 so legacy (v61 and the
		// pre-72 v28 Variants entry, neither IDA-backed for this field) omits it while
		// v72/v79/v83/84/87/95 keep the socketAddr wire unchanged.
		if (t.Region() == "GMS" && t.MajorVersion() >= 72) || t.Region() == "JMS" {
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
		// socketAddr int is v72+ (IDA v61 sub_564DC9@0x564dc9 omits it; v72
		// sub_5B1B25@0x5b1b25 adds getsockname->Encode4@0x5b1c92). Mirror of Encode.
		if (t.Region() == "GMS" && t.MajorVersion() >= 72) || t.Region() == "JMS" {
			m.socketAddr = r.ReadInt32()
		}
	}
}
