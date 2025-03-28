package macro

import (
	"atlas-skills/rest"
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
			r := router.PathPrefix("/characters/{characterId}/macros").Subrouter()
			r.HandleFunc("", rest.RegisterHandler(l)(si)("get_skill_macros", handleGetSkillMacros(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetSkillMacros(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				ms, err := GetByCharacterId(d.Context())(db)(characterId)
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
		})
	}
}
