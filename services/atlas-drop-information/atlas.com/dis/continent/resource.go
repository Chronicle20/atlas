package continent

import (
	"atlas-drops-information/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			r := router.PathPrefix("/continents/drops").Subrouter()
			r.HandleFunc("", registerGet("get_continent_drops", handleGetContinents)).Methods(http.MethodGet)
		}
	}
}

func handleGetContinents(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ms, err := GetAll(d.Logger())(d.Context())(d.DB())()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		res, err := model.SliceMap(Transform)(model.FixedProvider(ms))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[[]RestModel](d.Logger())(w)(c.ServerInformation())(res)
	}
}
