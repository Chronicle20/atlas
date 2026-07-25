package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const MerchantEmployeeDestroyWriter = "DestroyHiredMerchant"

// EmployeeDestroy is DESTROY_HIRED_MERCHANT — read by
// CEmployeePool::OnEmployeeLeaveField (v83 @0x510f20), which decodes a single u32
// employeeId and nothing else. Version-stable across all feature-bearing versions.
//
// packet-audit:fname CEmployeePool::OnEmployeeLeaveField
type EmployeeDestroy struct {
	employeeId uint32
}

func NewEmployeeDestroy(employeeId uint32) EmployeeDestroy {
	return EmployeeDestroy{employeeId: employeeId}
}

func (m EmployeeDestroy) EmployeeId() uint32 { return m.employeeId }
func (m EmployeeDestroy) Operation() string  { return MerchantEmployeeDestroyWriter }
func (m EmployeeDestroy) String() string {
	return fmt.Sprintf("employeeId [%d]", m.employeeId)
}

func (m EmployeeDestroy) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.employeeId)
		return w.Bytes()
	}
}

func (m *EmployeeDestroy) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.employeeId = r.ReadUint32()
	}
}
