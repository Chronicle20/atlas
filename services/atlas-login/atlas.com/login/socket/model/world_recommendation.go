package model

import (
	"context"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type Recommendation struct {
	worldId world.Id
	reason  string
}

func (r Recommendation) WorldId() world.Id {
	return r.worldId
}

func (r Recommendation) Reason() string {
	return r.reason
}

func NewWorldRecommendation(worldId world.Id, reason string) Recommendation {
	return Recommendation{worldId, reason}
}

func (r *Recommendation) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(uint32(r.WorldId()))
		w.WriteAsciiString(r.Reason())
		return w.Bytes()
	}
}
