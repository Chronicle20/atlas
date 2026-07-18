package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const DropPickUpHandle = "DropPickUpHandle"

// pickUpHasCRC reports whether the serverbound ITEM_PICKUP send carries the
// trailing client-CRC uint32. IDA-verified send sites: GMS v83
// (CWvsContext::SendDropPickUpRequest @0xa09118) and GMS v95 (@0x9d5d50) both
// Encode4(dwCliCrc) after the dropId, but GMS v79 (@0x954e9d) sends only
// fieldKey + updateTime + x + y + dropId with NO trailing crc. The crc was
// introduced in the GMS v80..v83 window; gate on major >= 83 (JMS185 carries it,
// pre-83 GMS — v79, v28 — does not).
func pickUpHasCRC(ctx context.Context) bool {
	t := tenant.MustFromContext(ctx)
	return t.MajorAtLeast(83)
}

// PickUp - CUser::SendDropPickUpRequest
// packet-audit:fname CWvsContext::SendDropPickUpRequest
type PickUp struct {
	fieldKey   byte
	updateTime uint32
	x          int16
	y          int16
	dropId     uint32
	crc        uint32
}

func (m PickUp) FieldKey() byte {
	return m.fieldKey
}

func (m PickUp) UpdateTime() uint32 {
	return m.updateTime
}

func (m PickUp) X() int16 {
	return m.x
}

func (m PickUp) Y() int16 {
	return m.y
}

func (m PickUp) DropId() uint32 {
	return m.dropId
}

func (m PickUp) CRC() uint32 {
	return m.crc
}

func (m PickUp) Operation() string {
	return DropPickUpHandle
}

func (m PickUp) String() string {
	return fmt.Sprintf("fieldKey [%d], updateTime [%d], x [%d], y [%d], dropId [%d], crc [%d]", m.fieldKey, m.updateTime, m.x, m.y, m.dropId, m.crc)
}

func (m PickUp) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.fieldKey)
		w.WriteInt(m.updateTime)
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		w.WriteInt(m.dropId)
		if pickUpHasCRC(ctx) {
			w.WriteInt(m.crc)
		}
		return w.Bytes()
	}
}

func (m *PickUp) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.fieldKey = r.ReadByte()
		m.updateTime = r.ReadUint32()
		m.x = r.ReadInt16()
		m.y = r.ReadInt16()
		m.dropId = r.ReadUint32()
		if pickUpHasCRC(ctx) {
			m.crc = r.ReadUint32()
		}
	}
}
