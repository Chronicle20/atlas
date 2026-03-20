package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterDistributeSpHandle = "CharacterDistributeSpHandle"

// DistributeSp - CWvsContext::SendIncSPMessage
type DistributeSp struct {
	updateTime uint32
	skillId    uint32
}

func (m DistributeSp) UpdateTime() uint32 { return m.updateTime }
func (m DistributeSp) SkillId() uint32    { return m.skillId }

func (m DistributeSp) Operation() string {
	return CharacterDistributeSpHandle
}

func (m DistributeSp) String() string {
	return fmt.Sprintf("updateTime [%d], skillId [%d]", m.updateTime, m.skillId)
}

func (m DistributeSp) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt(m.skillId)
		return w.Bytes()
	}
}

func (m *DistributeSp) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.skillId = r.ReadUint32()
	}
}
