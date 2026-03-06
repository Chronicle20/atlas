package writer

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const (
	UiLock = "UiLock"
)

func UiLockBody(enable bool, tAfterLeaveDirectionMode int32) packet.Encode {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		t := tenant.MustFromContext(ctx)
		return func(options map[string]interface{}) []byte {
			w.WriteBool(enable)
			if t.Region() == "GMS" && t.MajorVersion() >= 90 {
				w.WriteInt32(tAfterLeaveDirectionMode)
			}
			return w.Bytes()
		}
	}
}
