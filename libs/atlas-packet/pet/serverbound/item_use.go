package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const PetItemUseHandle = "PetItemUseHandle"

// packet-audit:fname CWvsContext::SendStatChangeItemUseRequestByPetQ
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

func (m ItemUse) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		if HasLeadingPetId(t) {
			w.WriteLong(m.petId) // absent on GMS v48 (single-pet; @0x70dc8d has no EncodeBuffer(petSN,8))
		}
		w.WriteBool(m.buffSkill)
		w.WriteInt(m.updateTime)
		w.WriteInt16(m.source)
		w.WriteInt(m.itemId)
		return w.Bytes()
	}
}

func (m *ItemUse) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if HasLeadingPetId(t) {
			m.petId = r.ReadUint64() // absent on GMS v48 (single-pet)
		}
		m.buffSkill = r.ReadBool()
		m.updateTime = r.ReadUint32()
		m.source = r.ReadInt16()
		m.itemId = r.ReadUint32()
	}
}
