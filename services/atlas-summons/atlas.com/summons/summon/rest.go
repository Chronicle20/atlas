package summon

import (
	"atlas-summons/rest"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const (
	getSummon = "get_summon"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/summons").Subrouter()
		r.HandleFunc("/{summonId}", rest.RegisterHandler(l)(si)(getSummon, handleGetSummonById)).Methods(http.MethodGet)
	}
}

func handleGetSummonById(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseSummonId(d.Logger(), func(summonId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context())
			m, err := p.GetById(summonId)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			res, err := model.Map(Transform)(model.FixedProvider(m))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res)
		}
	})
}
