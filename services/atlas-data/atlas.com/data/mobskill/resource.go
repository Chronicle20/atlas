package mobskill

import (
	"atlas-data/rest"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(db *gorm.DB) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)

			r := router.PathPrefix("/data/mob-skills").Subrouter()
			r.HandleFunc("", registerGet("get_mob_skills", handleGetMobSkillsRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{skillId}", registerGet("get_mob_skills_by_type", handleGetMobSkillsByTypeRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{skillId}/{level}", registerGet("get_mob_skill", handleGetMobSkillRequest(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetMobSkillsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			s := NewStorage(d.Logger(), db)
			res, err := s.GetAll(d.Context())
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to retrieve mob skills.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	}
}

func handleGetMobSkillsByTypeRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return parseMobSkillId(d.Logger(), func(skillId uint16) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s := NewStorage(d.Logger(), db)
				all, err := s.GetAll(d.Context())
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to retrieve mob skills.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				var results []RestModel
				for _, m := range all {
					if m.SkillId == skillId {
						results = append(results, m)
					}
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(results)
			}
		})
	}
}

func handleGetMobSkillRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return parseMobSkillId(d.Logger(), func(skillId uint16) http.HandlerFunc {
			return parseMobSkillLevel(d.Logger(), func(level uint16) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					s := NewStorage(d.Logger(), db)
					res, err := s.GetById(d.Context())(CompositeId(skillId, level))
					if err != nil {
						d.Logger().WithError(err).Debugf("Unable to locate mob skill %d level %d.", skillId, level)
						w.WriteHeader(http.StatusNotFound)
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

type mobSkillIdHandler func(skillId uint16) http.HandlerFunc

func parseMobSkillId(l logrus.FieldLogger, next mobSkillIdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		skillId, err := strconv.Atoi(vars["skillId"])
		if err != nil {
			l.WithError(err).Errorf("Error parsing skillId as uint16")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint16(skillId))(w, r)
	}
}

type mobSkillLevelHandler func(level uint16) http.HandlerFunc

func parseMobSkillLevel(l logrus.FieldLogger, next mobSkillLevelHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		level, err := strconv.Atoi(vars["level"])
		if err != nil {
			l.WithError(err).Errorf("Error parsing level as uint16")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint16(level))(w, r)
	}
}
