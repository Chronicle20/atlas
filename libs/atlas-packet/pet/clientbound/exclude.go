package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const PetExcludeResponseWriter = "PetExcludeResponse"

type ExcludeResponse struct {
	ownerId    uint32
	slot       int8
	petId      uint64
	excludeIds []uint32
}

func NewPetExcludeResponse(ownerId uint32, slot int8, petId uint64, excludeIds []uint32) ExcludeResponse {
	return ExcludeResponse{ownerId: ownerId, slot: slot, petId: petId, excludeIds: excludeIds}
}

func (m ExcludeResponse) Operation() string { return PetExcludeResponseWriter }
func (m ExcludeResponse) String() string {
	return fmt.Sprintf("ownerId [%d], slot [%d], excludes [%d]", m.ownerId, m.slot, len(m.excludeIds))
}

func (m ExcludeResponse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.ownerId)
		w.WriteInt8(m.slot)
		w.WriteLong(m.petId)
		w.WriteByte(byte(len(m.excludeIds)))
		for _, e := range m.excludeIds {
			w.WriteInt(e)
		}
		return w.Bytes()
	}
}

func (m *ExcludeResponse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.ownerId = r.ReadUint32()
		m.slot = r.ReadInt8()
		m.petId = r.ReadUint64()
		count := r.ReadByte()
		m.excludeIds = make([]uint32, count)
		for i := byte(0); i < count; i++ {
			m.excludeIds[i] = r.ReadUint32()
		}
	}
}
