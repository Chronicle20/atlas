package serverbound

import (
	"context"
	"fmt"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PetItemExcludeHandle = "PetItemExcludeHandle"

type ExcludeItem struct {
	petId   uint64
	itemIds []int32
}

func (m ExcludeItem) PetId() uint64 {
	return m.petId
}

func (m ExcludeItem) ItemIds() []int32 {
	return m.itemIds
}

func (m ExcludeItem) Operation() string {
	return PetItemExcludeHandle
}

func (m ExcludeItem) String() string {
	is := make([]string, len(m.itemIds))
	for i, id := range m.itemIds {
		is[i] = fmt.Sprintf("%d", id)
	}
	return fmt.Sprintf("petId [%d] items [%s]", m.petId, strings.Join(is, ","))
}

func (m ExcludeItem) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteLong(m.petId)
		w.WriteByte(byte(len(m.itemIds)))
		for _, id := range m.itemIds {
			w.WriteInt32(id)
		}
		return w.Bytes()
	}
}

func (m *ExcludeItem) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.petId = r.ReadUint64()
		count := r.ReadByte()
		m.itemIds = make([]int32, count)
		for i := range count {
			m.itemIds[i] = r.ReadInt32()
		}
	}
}
