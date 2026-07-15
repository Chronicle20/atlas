package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

func (m InfoRequest) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt(m.characterId)
		// The trailing petInfo bool (bCharacterInfoOfPet) was introduced at
		// GMS v61. Legacy GMS (<61) omits it: v48 CUser::SendCharacterInfoRequest
		// sub_71D059 @0x71d059 sends only Encode4(updateTime)+Encode4(charId)
		// (no third Encode1), whereas the v61 twin sub_845B68 @0x845b68 appends
		// Encode1(petInfo). task-113.
		if !(t.Region() == "GMS" && t.MajorVersion() < 61) {
			w.WriteBool(m.petInfo)
		}
		return w.Bytes()
	}
}

func (m *InfoRequest) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.characterId = r.ReadUint32()
		if !(t.Region() == "GMS" && t.MajorVersion() < 61) {
			m.petInfo = r.ReadBool()
		}
	}
}
