package shops

import (
	"atlas-npc/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			r := router.PathPrefix("/npcs/{npcId}/shop").Subrouter()
			r.HandleFunc("", registerGet("get_shop", handleGetShop)).Methods(http.MethodGet)
		}
	}
}

func handleGetShop(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseNpcId(d.Logger(), func(npcId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context(), d.DB())
			res, err := model.Map(Transform)(p.ByNpcIdProvider(npcId))()
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
