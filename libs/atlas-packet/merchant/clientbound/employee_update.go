package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MerchantEmployeeUpdateWriter = "UpdateHiredMerchant"

// EmployeeUpdate is UPDATE_HIRED_MERCHANT — read by
// CEmployeePool::OnEmployeeMiniRoomBalloon (v83 @0x510f7e): a u32 employeeId
// followed by the CEmployee::SetBalloon block. Refreshes the balloon (title /
// visitor state) of an already-spawned employee. Version-stable.
//
// packet-audit:fname CEmployeePool::OnEmployeeMiniRoomBalloon
type EmployeeUpdate struct {
	employeeId uint32
	balloon    Balloon
}

func NewEmployeeUpdate(employeeId uint32, balloon Balloon) EmployeeUpdate {
	return EmployeeUpdate{employeeId: employeeId, balloon: balloon}
}

func (m EmployeeUpdate) EmployeeId() uint32 { return m.employeeId }
func (m EmployeeUpdate) Balloon() Balloon   { return m.balloon }
func (m EmployeeUpdate) Operation() string  { return MerchantEmployeeUpdateWriter }
func (m EmployeeUpdate) String() string {
	return fmt.Sprintf("employeeId [%d], balloonType [%d]", m.employeeId, m.balloon.MiniRoomType())
}

func (m EmployeeUpdate) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.employeeId)
		w.WriteByteArray(m.balloon.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *EmployeeUpdate) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.employeeId = r.ReadUint32()
		m.balloon.Decode(l, ctx)(r, options)
	}
}
