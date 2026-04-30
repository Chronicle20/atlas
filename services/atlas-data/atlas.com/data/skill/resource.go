package skill

import (
	"atlas-data/rest"
	"net/http"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(db *gorm.DB) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)

			r := router.PathPrefix("/data/skills").Subrouter()
			r.HandleFunc("", registerGet("search_skills", handleSearchSkillsRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{skillId}", registerGet("get_skill", handleGetReactorRequest(db))).Methods(http.MethodGet)
		}
	}
}

func handleSearchSkillsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			idParams := query["ids"]
			nameQuery := query.Get("name")

			if len(idParams) == 0 && nameQuery == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			s := NewStorage(d.Logger(), db)
			allSkills, err := s.GetAll(d.Context())
			if err != nil {
				d.Logger().WithError(err).Debugf("Unable to get all skills.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			results := make([]RestModel, 0)

			if len(idParams) > 0 {
				idSet := make(map[uint32]struct{})
				for _, raw := range idParams {
					for _, part := range strings.Split(raw, ",") {
						part = strings.TrimSpace(part)
						if part == "" {
							continue
						}
						id, err := strconv.ParseUint(part, 10, 32)
						if err != nil {
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						idSet[uint32(id)] = struct{}{}
					}
				}
				for _, sk := range allSkills {
					if _, ok := idSet[sk.Id]; ok {
						results = append(results, sk)
					}
				}
			} else {
				nameQueryLower := strings.ToLower(nameQuery)
				for _, sk := range allSkills {
					if strings.Contains(strings.ToLower(sk.Name), nameQueryLower) {
						results = append(results, sk)
						if len(results) >= 10 {
							break
						}
					}
				}
			}

			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(results)
		}
	}
}

func handleGetReactorRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseSkillId(d.Logger(), func(skillId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s := NewStorage(d.Logger(), db)
				res, err := s.GetById(d.Context())(strconv.Itoa(int(skillId)))
				if err != nil {
					d.Logger().WithError(err).Debugf("Unable to locate skill %d.", skillId)
					w.WriteHeader(http.StatusNotFound)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}
