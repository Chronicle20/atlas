package model

import (
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Recommendation struct {
	worldId byte
	reason  string
}

func (r Recommendation) WorldId() byte {
	return r.worldId
}

func (r Recommendation) Reason() string {
	return r.reason
}

func NewWorldRecommendation(worldId byte, reason string) Recommendation {
	return Recommendation{worldId, reason}
}

func (r *Recommendation) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt(uint32(r.WorldId()))
		w.WriteAsciiString(r.Reason())
	}
}
