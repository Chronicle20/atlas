package recipe

import (
	"atlas-npc-conversations/rest"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerHandler := rest.RegisterHandler(l)(db)(si)
			router.HandleFunc("/items/{itemId}/recipes", registerHandler("get_recipes_by_item", GetByItemHandler)).Methods(http.MethodGet)
			router.HandleFunc("/npcs/{npcId}/recipes", registerHandler("get_recipes_by_npc", GetByNpcHandler)).Methods(http.MethodGet)
		}
	}
}

func GetByItemHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseItemId(d.Logger(), func(itemId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			mp := NewProcessor(d.Logger(), d.Context(), d.DB()).ByItemIdProvider(itemId)
			rms, err := model.SliceMap(transformProvider)(mp)(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Listing recipes by itemId=%d", itemId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rms)
		}
	})
}

func GetByNpcHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseNpcId(d.Logger(), func(npcId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			mp := NewProcessor(d.Logger(), d.Context(), d.DB()).ByNpcIdProvider(npcId)
			rms, err := model.SliceMap(transformProvider)(mp)(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Listing recipes by npcId=%d", npcId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rms)
		}
	})
}

func transformProvider(m Model) (RestModel, error) { return Transform(m), nil }
