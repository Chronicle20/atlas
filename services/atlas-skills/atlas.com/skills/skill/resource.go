package skill

import (
	"atlas-skills/rest"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/characters/{characterId}/skills").Subrouter()
			r.HandleFunc("", rest.RegisterHandler(l)(si)("get_skills", handleGetSkills(db))).Methods(http.MethodGet)
			r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(si)("create_skill", handleRequestCreateSkill(db))).Methods(http.MethodPost)
			r.HandleFunc("/{skillId}", rest.RegisterHandler(l)(si)("get_skill", handleGetSkill(db))).Methods(http.MethodGet)
			r.HandleFunc("/{skillId}", rest.RegisterInputHandler[RestModel](l)(si)("update_skill", handleRequestUpdateSkill(db))).Methods(http.MethodPatch)
		}
	}
}

func handleGetSkills(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				mp := NewProcessor(d.Logger(), d.Context(), db).ByCharacterIdProvider(characterId)
				res, err := model.SliceMap(Transform)(mp)(model.ParallelMap())()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}

func handleRequestCreateSkill(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, i RestModel) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := NewProcessor(d.Logger(), d.Context(), db).RequestCreate(uuid.New(), world.Id(0), characterId, i.Id, i.Level, i.MasterLevel, i.Expiration)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusAccepted)
			}
		})
	}
}

func handleGetSkill(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return rest.ParseSkillId(d.Logger(), func(skillId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					mp := NewProcessor(d.Logger(), d.Context(), db).ByIdProvider(characterId, skillId)
					res, err := model.Map(Transform)(mp)()
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
		})
	}
}

func handleRequestUpdateSkill(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, i RestModel) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return rest.ParseSkillId(d.Logger(), func(skillId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					err := NewProcessor(d.Logger(), d.Context(), db).RequestUpdate(uuid.New(), world.Id(0), characterId, skillId, i.Level, i.MasterLevel, i.Expiration)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusAccepted)
				}
			})
		})
	}
}
