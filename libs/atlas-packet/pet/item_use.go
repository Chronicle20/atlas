package pet

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PetItemUseHandle = "PetItemUseHandle"

type ItemUse struct {
	petId      uint64
	buffSkill  bool
	updateTime uint32
	source     int16
	itemId     uint32
}

func (m ItemUse) PetId() uint64 {
	return m.petId
}

func (m ItemUse) BuffSkill() bool {
	return m.buffSkill
}

func (m ItemUse) UpdateTime() uint32 {
	return m.updateTime
}

func (m ItemUse) Source() int16 {
	return m.source
}

func (m ItemUse) ItemId() uint32 {
	return m.itemId
}

func (m ItemUse) Operation() string {
	return PetItemUseHandle
}

func (m ItemUse) String() string {
	return fmt.Sprintf("petId [%d] buffSkill [%t] updateTime [%d] source [%d] itemId [%d]", m.petId, m.buffSkill, m.updateTime, m.source, m.itemId)
}

func (m ItemUse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteLong(m.petId)
		w.WriteBool(m.buffSkill)
		w.WriteInt(m.updateTime)
		w.WriteInt16(m.source)
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *ItemUse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.petId = r.ReadUint64()
		m.buffSkill = r.ReadBool()
		m.updateTime = r.ReadUint32()
		m.source = r.ReadInt16()
		m.itemId = r.ReadUint32()
	}
}
