package login

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const WorldSelectHandle = "WorldSelectHandle"

// ServerSelect - CLogin::SendWorldSelectPacket
type ServerSelect struct {
	worldId world.Id
}

func (m ServerSelect) WorldId() world.Id {
	return m.worldId
}

func (m ServerSelect) Operation() string {
	return WorldSelectHandle
}

func (m ServerSelect) String() string {
	return fmt.Sprintf("worldId [%d]", m.worldId)
}

func (m ServerSelect) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(m.worldId))
		return w.Bytes()
	}
}

func (m *ServerSelect) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.worldId = world.Id(r.ReadByte())
	}
}
