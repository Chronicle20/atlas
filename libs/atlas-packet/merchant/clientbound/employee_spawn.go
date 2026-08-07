package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const MerchantEmployeeSpawnWriter = "SpawnHiredMerchant"

// EmployeeSpawn is SPAWN_HIRED_MERCHANT — the field appearance of a hired
// merchant, read by CEmployeePool::OnEmployeeEnterField. The wire layout is
// byte-identical across every version that has the feature (gms v61/72/79/83/84/
// 87/95, jms v185); gms v48 has no hired-merchant feature, so this packet is never
// routed there. No version gate on field order is required.
//
// packet-audit:fname CEmployeePool::OnEmployeeEnterField
type EmployeeSpawn struct {
	employeeId uint32
	templateId uint32
	x          int16
	y          int16
	foothold   int16
	ownerName  string
	balloon    Balloon
}

func NewEmployeeSpawn(employeeId uint32, templateId uint32, x int16, y int16, foothold int16, ownerName string, balloon Balloon) EmployeeSpawn {
	return EmployeeSpawn{
		employeeId: employeeId,
		templateId: templateId,
		x:          x,
		y:          y,
		foothold:   foothold,
		ownerName:  ownerName,
		balloon:    balloon,
	}
}

func (m EmployeeSpawn) EmployeeId() uint32 { return m.employeeId }
func (m EmployeeSpawn) TemplateId() uint32 { return m.templateId }
func (m EmployeeSpawn) X() int16           { return m.x }
func (m EmployeeSpawn) Y() int16           { return m.y }
func (m EmployeeSpawn) Foothold() int16    { return m.foothold }
func (m EmployeeSpawn) OwnerName() string  { return m.ownerName }
func (m EmployeeSpawn) Balloon() Balloon   { return m.balloon }
func (m EmployeeSpawn) Operation() string  { return MerchantEmployeeSpawnWriter }
func (m EmployeeSpawn) String() string {
	return fmt.Sprintf("employeeId [%d], templateId [%d], x [%d], y [%d], foothold [%d], ownerName [%s]", m.employeeId, m.templateId, m.x, m.y, m.foothold, m.ownerName)
}

// Encode mirrors CEmployeePool::OnEmployeeEnterField -> CEmployee::Init (v83
// @0x50d56c: Decode2 x, Decode2 y, Decode2 fh, DecodeStr ownerName) -> SetBalloon.
func (m EmployeeSpawn) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.employeeId)
		w.WriteInt(m.templateId)
		w.WriteInt16(m.x)
		w.WriteInt16(m.y)
		w.WriteInt16(m.foothold)
		w.WriteAsciiString(m.ownerName)
		w.WriteByteArray(m.balloon.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *EmployeeSpawn) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.employeeId = r.ReadUint32()
		m.templateId = r.ReadUint32()
		m.x = r.ReadInt16()
		m.y = r.ReadInt16()
		m.foothold = r.ReadInt16()
		m.ownerName = r.ReadAsciiString()
		m.balloon.Decode(l, ctx)(r, options)
	}
}
