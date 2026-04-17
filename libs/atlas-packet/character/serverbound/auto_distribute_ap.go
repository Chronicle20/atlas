package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterAutoDistributeApHandle = "CharacterAutoDistributeApHandle"

type DistributeEntry struct {
	Flag  uint32
	Value uint32
}

// AutoDistributeAp - CWvsContext::SendAutoIncAPMessage
type AutoDistributeAp struct {
	updateTime  uint32
	nValue      uint32
	distributes []DistributeEntry
}

func (m AutoDistributeAp) UpdateTime() uint32           { return m.updateTime }
func (m AutoDistributeAp) NValue() uint32               { return m.nValue }
func (m AutoDistributeAp) Distributes() []DistributeEntry { return m.distributes }

func (m AutoDistributeAp) Operation() string {
	return CharacterAutoDistributeApHandle
}

func (m AutoDistributeAp) String() string {
	return fmt.Sprintf("updateTime [%d], nValue [%d], distributes [%d]", m.updateTime, m.nValue, len(m.distributes))
}

func (m AutoDistributeAp) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt(m.nValue)
		for _, d := range m.distributes {
			w.WriteInt(d.Flag)
			w.WriteInt(d.Value)
		}
		return w.Bytes()
	}
}

func (m *AutoDistributeAp) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.nValue = r.ReadUint32()
		m.distributes = make([]DistributeEntry, 0)
		for r.Available() >= 8 {
			flag := r.ReadUint32()
			value := r.ReadUint32()
			m.distributes = append(m.distributes, DistributeEntry{Flag: flag, Value: value})
		}
	}
}
