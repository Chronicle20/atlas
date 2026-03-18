package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterChairPortableHandle = "CharacterChairPortableHandle"

// ChairPortable - CUser::SendSitOnPortableChair
type ChairPortable struct {
	itemId uint32
}

func (m ChairPortable) ItemId() uint32 {
	return m.itemId
}

func (m ChairPortable) Operation() string {
	return CharacterChairPortableHandle
}

func (m ChairPortable) String() string {
	return fmt.Sprintf("itemId [%d]", m.itemId)
}

func (m ChairPortable) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *ChairPortable) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.itemId = r.ReadUint32()
	}
}
