package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterItemUseScrollHandle = "CharacterItemUseScrollHandle"

// ScrollUse - CUser::SendScrollUseRequest
type ScrollUse struct {
	updateTime      uint32
	scrollSlot      int16
	equipSlot       int16
	bWhiteScroll    int16
	legendarySpirit bool
}

func (m ScrollUse) UpdateTime() uint32   { return m.updateTime }
func (m ScrollUse) ScrollSlot() int16    { return m.scrollSlot }
func (m ScrollUse) EquipSlot() int16     { return m.equipSlot }
func (m ScrollUse) WhiteScroll() bool    { return (m.bWhiteScroll & 2) == 2 }
func (m ScrollUse) LegendarySpirit() bool { return m.legendarySpirit }

func (m ScrollUse) Operation() string {
	return CharacterItemUseScrollHandle
}

func (m ScrollUse) String() string {
	return fmt.Sprintf("updateTime [%d], scrollSlot [%d], equipSlot [%d], whiteScroll [%t], legendarySpirit [%t]", m.updateTime, m.scrollSlot, m.equipSlot, m.WhiteScroll(), m.legendarySpirit)
}

func (m ScrollUse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt16(m.scrollSlot)
		w.WriteInt16(m.equipSlot)
		w.WriteInt16(m.bWhiteScroll)
		w.WriteBool(m.legendarySpirit)
		return w.Bytes()
	}
}

func (m *ScrollUse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.scrollSlot = r.ReadInt16()
		m.equipSlot = r.ReadInt16()
		m.bWhiteScroll = r.ReadInt16()
		m.legendarySpirit = r.ReadBool()
	}
}
