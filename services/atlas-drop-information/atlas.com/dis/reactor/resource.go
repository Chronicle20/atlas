package reactor

import (
	"atlas-drops-information/reactor/drop"
	"atlas-drops-information/rest"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			r := router.PathPrefix("/reactors/{reactorId}/drops").Subrouter()
			r.HandleFunc("", registerGet("get_reactor_drops", handleGetReactorDrops)).Methods(http.MethodGet)
		}
	}
}

type ReactorIdHandler func(reactorId uint32) http.HandlerFunc

func ParseReactorId(l logrus.FieldLogger, next ReactorIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reactorId, err := strconv.Atoi(mux.Vars(r)["reactorId"])
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse reactorId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(reactorId))(w, r)
	}
}

func handleGetReactorDrops(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return ParseReactorId(d.Logger(), func(reactorId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ms, err := drop.NewProcessor(d.Logger(), d.Context(), d.DB()).GetForReactor(reactorId)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving drops for reactor [%d].", reactorId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Transform to reactor rest model with included drops
			res, err := Transform(reactorId, ms)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}
