package skill

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

func handleRequestCreateSkill(_ *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, i RestModel) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := RequestCreate(d.Logger())(d.Context())(characterId, i.Id, i.Level, i.MasterLevel, i.Expiration)
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
					m, err := GetById(d.Context())(db)(characterId, skillId)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
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
		})
	}
}

func handleRequestUpdateSkill(_ *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, i RestModel) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return rest.ParseSkillId(d.Logger(), func(skillId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					err := RequestUpdate(d.Logger())(d.Context())(characterId, skillId, i.Level, i.MasterLevel, i.Expiration)
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
