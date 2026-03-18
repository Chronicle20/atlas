package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ServerListRecommendationsWriter = "ServerListRecommendations"

type ServerListRecommendations struct {
	recommendations []model.WorldRecommendation
}

func NewServerListRecommendations(recommendations []model.WorldRecommendation) ServerListRecommendations {
	return ServerListRecommendations{recommendations: recommendations}
}

func (m ServerListRecommendations) Recommendations() []model.WorldRecommendation {
	return m.recommendations
}
func (m ServerListRecommendations) Operation() string { return ServerListRecommendationsWriter }
func (m ServerListRecommendations) String() string {
	return fmt.Sprintf("recommendations [%d]", len(m.recommendations))
}

func (m ServerListRecommendations) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(len(m.recommendations)))
		for _, x := range m.recommendations {
			x.Write(w)
		}
		return w.Bytes()
	}
}

func (m *ServerListRecommendations) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		count := r.ReadByte()
		m.recommendations = make([]model.WorldRecommendation, count)
		for i := byte(0); i < count; i++ {
			var wr model.WorldRecommendation
			wr.Read(r)
			m.recommendations[i] = wr
		}
	}
}
