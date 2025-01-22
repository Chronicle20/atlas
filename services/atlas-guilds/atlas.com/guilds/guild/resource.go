package guild

import (
	"atlas-guilds/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

const (
	GetGuilds = "get_guilds"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)
			r := router.PathPrefix("/guilds").Subrouter()
			r.HandleFunc("/", registerGet(GetGuilds, handleGetGuilds(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetGuilds(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			gs, err := GetSlice(d.Logger())(d.Context())(db)()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := model.SliceMap(Transform)(model.FixedProvider(gs))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[[]RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	}
}
