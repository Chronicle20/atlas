package monster

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MonsterControlWriter = "ControlMonster"

type ControlType int8

const (
	ControlTypeReset         ControlType = 0
	ControlTypeActiveInit    ControlType = 1
	ControlTypeActiveRequest ControlType = 2
	ControlTypeActivePerm0   ControlType = 3
	ControlTypeActivePerm1   ControlType = 4
	ControlTypePassive       ControlType = -1
	ControlTypePassive0      ControlType = -2
	ControlTypePassive1      ControlType = -3
)

type Control struct {
	controlType ControlType
	uniqueId    uint32
	monsterId   uint32
	monster     model.MonsterModel
}

func NewMonsterControl(controlType ControlType, uniqueId uint32, monsterId uint32, monster model.MonsterModel) Control {
	return Control{
		controlType: controlType,
		uniqueId:    uniqueId,
		monsterId:   monsterId,
		monster:     monster,
	}
}

func (m Control) ControlTypeValue() ControlType    { return m.controlType }
func (m Control) UniqueId() uint32                  { return m.uniqueId }
func (m Control) MonsterId() uint32                 { return m.monsterId }
func (m Control) Monster() model.MonsterModel       { return m.monster }
func (m Control) Operation() string                 { return MonsterControlWriter }
func (m Control) String() string {
	return fmt.Sprintf("controlType [%d], uniqueId [%d], monsterId [%d]", m.controlType, m.uniqueId, m.monsterId)
}

func (m Control) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt8(int8(m.controlType))
		w.WriteInt(m.uniqueId)
		if m.controlType > ControlTypeReset {
			w.WriteByte(5)
			w.WriteInt(m.monsterId)
			w.WriteByteArray(m.monster.Encode(l, ctx)(options))
		}
		return w.Bytes()
	}
}

func (m *Control) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.controlType = ControlType(r.ReadInt8())
		m.uniqueId = r.ReadUint32()
		if m.controlType > ControlTypeReset {
			_ = r.ReadByte() // always 5
			m.monsterId = r.ReadUint32()
			m.monster.Decode(l, ctx)(r, options)
		}
	}
}
