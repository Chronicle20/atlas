package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const ServerStatusHandle = "ServerStatusHandle"

type ServerStatusRequest struct {
	worldId world.Id
}

func (m ServerStatusRequest) WorldId() world.Id {
	return m.worldId
}

func (m ServerStatusRequest) Operation() string {
	return ServerStatusHandle
}

func (m ServerStatusRequest) String() string {
	return fmt.Sprintf("worldId [%d]", m.worldId)
}

func (m ServerStatusRequest) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			w.WriteShort(uint16(m.worldId))
		} else {
			w.WriteByte(byte(m.worldId))
		}
		return w.Bytes()
	}
}

func (m *ServerStatusRequest) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			m.worldId = world.Id(r.ReadUint16())
		} else {
			m.worldId = world.Id(r.ReadByte())
		}
	}
}
