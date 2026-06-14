package clientbound

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ForcedMapEquipWriter = "ForcedMapEquip"

// ForcedMapEquip is the clientbound CField::OnFieldSpecificData packet.
// The v83 handler decodes no fields (the client vtable-forwards to field-specific
// data handling), so the wire payload is empty.
type ForcedMapEquip struct {
}

func NewForcedMapEquip() ForcedMapEquip {
	return ForcedMapEquip{}
}

func (m ForcedMapEquip) Operation() string { return ForcedMapEquipWriter }
func (m ForcedMapEquip) String() string {
	return "ForcedMapEquip"
}

func (m ForcedMapEquip) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		return w.Bytes()
	}
}

func (m *ForcedMapEquip) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
