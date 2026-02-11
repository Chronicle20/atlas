package writer

import (
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
)

const (
	UiDisable = "UiDisable"
)

func UiDisableBody(_ tenant.Model) func(enable bool) BodyProducer {
	return func(enable bool) BodyProducer {
		return func(w *response.Writer, options map[string]interface{}) []byte {
			w.WriteBool(enable)
			return w.Bytes()
		}
	}
}
