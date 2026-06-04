package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterLoggedInHandle = "CharacterLoggedInHandle"

// ChannelConnect - CClientSocket::OnConnect
type ChannelConnect struct {
	characterId uint32
	machineId   []byte
	gm          bool
	unknown1    bool
	unknown2    uint64
}

func (m ChannelConnect) CharacterId() uint32 {
	return m.characterId
}

func (m ChannelConnect) MachineId() []byte {
	return m.machineId
}

func (m ChannelConnect) Gm() bool {
	return m.gm
}

func (m ChannelConnect) Unknown1() bool {
	return m.unknown1
}

func (m ChannelConnect) Unknown2() uint64 {
	return m.unknown2
}

func (m ChannelConnect) Operation() string {
	return CharacterLoggedInHandle
}

func (m ChannelConnect) String() string {
	return fmt.Sprintf("characterId [%d], machineId [%s], gm [%t], unknown1 [%t], unknown2 [%d]", m.characterId, m.machineId, m.gm, m.unknown1, m.unknown2)
}

func (m ChannelConnect) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.CharacterId())
		w.WriteByteArray(m.MachineId())
		// JMS v185 CClientSocket::OnConnect (non-login branch @ 0x4b051f) encodes
		// the gm/dummy1 field as Encode2 (uint16) while GMS uses Encode1 (1 byte).
		// JMS IDA: COutPacket::Encode2(v47, TSingleton<CConfig>::ms_pInstance->dummy1)
		if t.Region() == "JMS" {
			w.WriteShort(uint16(boolToUint8(m.Gm())))
		} else {
			w.WriteBool(m.Gm())
		}
		w.WriteBool(m.Unknown1())
		w.WriteLong(m.Unknown2())
		return w.Bytes()
	}
}

func (m *ChannelConnect) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.machineId = r.ReadBytes(16)
		// JMS v185 sends gm/dummy1 as a 2-byte uint16; GMS sends 1 byte.
		if t.Region() == "JMS" {
			m.gm = r.ReadUint16() != 0
		} else {
			m.gm = r.ReadBool()
		}
		m.unknown1 = r.ReadBool()
		m.unknown2 = r.ReadUint64()
	}
}

func boolToUint8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}
