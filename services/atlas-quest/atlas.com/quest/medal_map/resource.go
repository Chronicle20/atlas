package medal_map

import (
	"atlas-quest/rest"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

const RecordMedalMap = "record_medal_map"

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/characters/{characterId}/quests/{questId}/medal-maps").Subrouter()
			r.HandleFunc("", rest.RegisterInputHandler[PostRestModel](l)(db)(si)(RecordMedalMap, handleRecordMedalMap)).Methods(http.MethodPost)
		}
	}
}

func handleRecordMedalMap(d *rest.HandlerDependency, c *rest.HandlerContext, input PostRestModel) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return rest.ParseQuestId(d.Logger(), func(questId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				result, err := NewProcessor(d.Logger(), d.Context(), d.DB()).Record(characterId, questId, _map.Id(input.MapId))
				if err != nil {
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				res, err := model.Map(Transform)(model.FixedProvider(result))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				if result.NewlyRecorded {
					w.WriteHeader(http.StatusCreated)
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	})
}
