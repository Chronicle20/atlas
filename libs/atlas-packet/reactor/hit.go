package reactor

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ReactorHitHandle = "ReactorHitHandle"

// Hit - CReactorPool::OnHitReactor
type Hit struct {
	oid          uint32
	isSkill      bool
	dwHitOption  uint32
	delay        uint16
	skillId      uint32
}

func (m Hit) Oid() uint32         { return m.oid }
func (m Hit) IsSkill() bool       { return m.isSkill }
func (m Hit) DwHitOption() uint32 { return m.dwHitOption }
func (m Hit) Delay() uint16       { return m.delay }
func (m Hit) SkillId() uint32     { return m.skillId }

func (m Hit) Operation() string {
	return ReactorHitHandle
}

func (m Hit) String() string {
	return fmt.Sprintf("oid [%d], isSkill [%t], dwHitOption [%d], delay [%d], skillId [%d]", m.oid, m.isSkill, m.dwHitOption, m.delay, m.skillId)
}

func (m Hit) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.oid)
		if m.isSkill {
			w.WriteInt(1)
		} else {
			w.WriteInt(0)
		}
		w.WriteInt(m.dwHitOption)
		w.WriteShort(m.delay)
		w.WriteInt(m.skillId)
		return w.Bytes()
	}
}

func (m *Hit) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.oid = r.ReadUint32()
		m.isSkill = r.ReadUint32() == 1
		m.dwHitOption = r.ReadUint32()
		m.delay = r.ReadUint16()
		m.skillId = r.ReadUint32()
	}
}
