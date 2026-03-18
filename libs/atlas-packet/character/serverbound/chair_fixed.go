package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterChairInteractionHandle = "CharacterChairInteractionHandle"

// ChairFixed - CUser::SendSitOnMapSeat
type ChairFixed struct {
	chairId int16
}

func (m ChairFixed) ChairId() int16 {
	return m.chairId
}

func (m ChairFixed) Operation() string {
	return CharacterChairInteractionHandle
}

func (m ChairFixed) String() string {
	return fmt.Sprintf("chairId [%d]", m.chairId)
}

func (m ChairFixed) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt16(m.chairId)
		return w.Bytes()
	}
}

func (m *ChairFixed) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.chairId = r.ReadInt16()
	}
}
