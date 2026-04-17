package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SelectWorldWriter = "SelectWorld"

type SelectWorld struct {
	worldId uint32
}

func NewSelectWorld(worldId uint32) SelectWorld {
	return SelectWorld{worldId: worldId}
}

func (m SelectWorld) WorldId() uint32    { return m.worldId }
func (m SelectWorld) Operation() string  { return SelectWorldWriter }
func (m SelectWorld) String() string     { return fmt.Sprintf("worldId [%d]", m.worldId) }

func (m SelectWorld) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.worldId)
		return w.Bytes()
	}
}

func (m *SelectWorld) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.worldId = r.ReadUint32()
	}
}
