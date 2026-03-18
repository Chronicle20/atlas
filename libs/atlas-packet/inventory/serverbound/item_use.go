package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const (
	CharacterItemUseHandle           = "CharacterItemUseHandle"
	CharacterItemUseTownScrollHandle = "CharacterItemUseTownScrollHandle"
	CharacterItemUseSummonBagHandle  = "CharacterItemUseSummonBagHandle"
)

// ItemUse - CUser::SendItemUseRequest
type ItemUse struct {
	operation  string
	updateTime uint32
	source     int16
	itemId     uint32
}

func NewItemUse(operation string) ItemUse {
	return ItemUse{operation: operation}
}

func (m ItemUse) UpdateTime() uint32 { return m.updateTime }
func (m ItemUse) Source() int16      { return m.source }
func (m ItemUse) ItemId() uint32     { return m.itemId }

func (m ItemUse) Operation() string {
	return m.operation
}

func (m ItemUse) String() string {
	return fmt.Sprintf("updateTime [%d], source [%d], itemId [%d]", m.updateTime, m.source, m.itemId)
}

func (m ItemUse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.updateTime)
		w.WriteInt16(m.source)
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *ItemUse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.updateTime = r.ReadUint32()
		m.source = r.ReadInt16()
		m.itemId = r.ReadUint32()
	}
}
