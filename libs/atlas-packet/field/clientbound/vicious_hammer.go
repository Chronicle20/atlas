package clientbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ViciousHammerWriter = "ViciousHammer"

// ViciousHammer has an empty body. CField::OnItemUpgrade is a vtable forwarder
// (delegates the opcode to the item-upgrade dialog) and reads no wire fields of
// its own. The op is absent from the jms registry.
type ViciousHammer struct {
}

func NewViciousHammer() ViciousHammer {
	return ViciousHammer{}
}

func (m ViciousHammer) Operation() string { return ViciousHammerWriter }
func (m ViciousHammer) String() string {
	return "ViciousHammer"
}

func (m ViciousHammer) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

func (m *ViciousHammer) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
