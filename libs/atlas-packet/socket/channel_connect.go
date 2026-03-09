package socket

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
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

func (m ChannelConnect) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.CharacterId())
		w.WriteByteArray(m.MachineId())
		w.WriteBool(m.Gm())
		w.WriteBool(m.Unknown1())
		w.WriteLong(m.Unknown2())
		return w.Bytes()
	}
}

func (m *ChannelConnect) Decode(l logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.machineId = r.ReadBytes(16)
		m.gm = r.ReadBool()
		m.unknown1 = r.ReadBool()
		m.unknown2 = r.ReadUint64()
	}
}
