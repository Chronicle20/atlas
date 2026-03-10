package model

import (
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
)

type WorldRecommendation struct {
	worldId world.Id
	reason  string
}

func NewWorldRecommendation(worldId world.Id, reason string) WorldRecommendation {
	return WorldRecommendation{worldId: worldId, reason: reason}
}

func (m WorldRecommendation) WorldId() world.Id { return m.worldId }
func (m WorldRecommendation) Reason() string     { return m.reason }

func (m WorldRecommendation) Write(w *response.Writer) {
	w.WriteInt(uint32(m.worldId))
	w.WriteAsciiString(m.reason)
}

func (m *WorldRecommendation) Read(r *request.Reader) {
	m.worldId = world.Id(r.ReadUint32())
	m.reason = r.ReadAsciiString()
}
