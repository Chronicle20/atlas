package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterDistributeApHandle = "CharacterDistributeApHandle"

// DistributeAp - CWvsContext::SendIncAPMessage
type DistributeAp struct {
	updateTime uint32
	dwFlag     uint32
}

func (m DistributeAp) UpdateTime() uint32 { return m.updateTime }
func (m DistributeAp) DwFlag() uint32     { return m.dwFlag }

func (m DistributeAp) Operation() string {
	return CharacterDistributeApHandle
}

func (m DistributeAp) String() string {
	return fmt.Sprintf("updateTime [%d], dwFlag [%d]", m.updateTime, m.dwFlag)
}

func (m DistributeAp) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt(m.dwFlag)
		return w.Bytes()
	}
}

func (m *DistributeAp) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.dwFlag = r.ReadUint32()
	}
}
