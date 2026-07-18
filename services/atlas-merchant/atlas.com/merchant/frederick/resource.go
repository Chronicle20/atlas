package frederick

import (
	"atlas-merchant/rest"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

func InitializeRoutes(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerHandler := rest.RegisterHandler(l)(si)

			cr := router.PathPrefix("/characters/{characterId}").Subrouter()
			cr.HandleFunc("/frederick", registerHandler("get_character_frederick", handleGetCharacterFrederick(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetCharacterFrederick(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				hasPending, err := NewProcessor(d.Logger(), d.Context(), db).HasPending(characterId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Retrieving frederick status for character [%d].", characterId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				res, err := TransformStatus(characterId, hasPending)
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[StatusRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}
