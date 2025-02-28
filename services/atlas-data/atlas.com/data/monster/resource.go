package monster

import (
	"atlas-data/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)

		r := router.PathPrefix("/data/monsters").Subrouter()
		r.HandleFunc("/{monsterId}", registerGet("get_monster", handleGetMonsterRequest)).Methods(http.MethodGet)
		r.HandleFunc("/{monsterId}/loseItems", registerGet("get_monster_lose_items", handleGetMonsterLoseItemsRequest)).Methods(http.MethodGet)
	}
}

func handleGetMonsterRequest(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseMonsterId(d.Logger(), func(monsterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := GetById(d.Context())(monsterId)
			if err != nil {
				d.Logger().WithError(err).Debugf("Unable to locate monster %d.", monsterId)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			res, err := model.Map(Transform)(model.FixedProvider(m))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}

func handleGetMonsterLoseItemsRequest(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseMonsterId(d.Logger(), func(monsterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := GetById(d.Context())(monsterId)
			if err != nil {
				d.Logger().WithError(err).Debugf("Unable to locate monster %d.", monsterId)
				w.WriteHeader(http.StatusNotFound)
				return
			}

			res, err := model.SliceMap(TransformLoseItem)(model.FixedProvider(m.loseItems))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[[]loseItem](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}
