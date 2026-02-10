package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
)

const (
	UiLock = "UiLock"
)

func UiLockBody(t tenant.Model) func(enable bool, tAfterLeaveDirectionMode int32) BodyProducer {
	return func(enable bool, tAfterLeaveDirectionMode int32) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteBool(enable)
			if t.Region() == "GMS" && t.MajorVersion() >= 90 {
				w.WriteInt32(tAfterLeaveDirectionMode)
			}
			return w.Bytes()
		}
	}
}
