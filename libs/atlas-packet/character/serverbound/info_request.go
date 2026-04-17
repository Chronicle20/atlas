package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterInfoRequestHandle = "CharacterInfoRequestHandle"

// InfoRequest - CUser::SendCharacterInfoRequest
type InfoRequest struct {
	updateTime  uint32
	characterId uint32
	petInfo     bool
}

func (m InfoRequest) UpdateTime() uint32 {
	return m.updateTime
}

func (m InfoRequest) CharacterId() uint32 {
	return m.characterId
}

func (m InfoRequest) PetInfo() bool {
	return m.petInfo
}

func (m InfoRequest) Operation() string {
	return CharacterInfoRequestHandle
}

func (m InfoRequest) String() string {
	return fmt.Sprintf("updateTime [%d], characterId [%d], petInfo [%t]", m.updateTime, m.characterId, m.petInfo)
}

func (m InfoRequest) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt(m.characterId)
		w.WriteBool(m.petInfo)
		return w.Bytes()
	}
}

func (m *InfoRequest) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.characterId = r.ReadUint32()
		m.petInfo = r.ReadBool()
	}
}
