package job

import (
	"atlas-data/rest"
	"atlas-data/skill"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

func InitResource(db *gorm.DB) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)

			r := router.PathPrefix("/data/jobs").Subrouter()
			r.HandleFunc("", registerGet("get_jobs", handleGetJobsRequest(db))).Methods(http.MethodGet)
			r.HandleFunc("/{jobId}/skills",
				registerGet("get_job_skills", handleGetJobSkills(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetJobSkills(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseJobId(d.Logger(), func(jobId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, ok := NewProcessor(d.Logger(), d.Context(), db).GetSkillsForJob(jobId)
				if !ok {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(m)
			}
		})
	}
}

// includesSkills reports whether the request asked for the skills relationship
// to be materialized. Neither server.MarshalResponse nor
// server.MarshalPaginatedResponse knows about `include` —
// jsonapi.FilterSparseFields handles `fields[type]` only — so the handler
// parses it itself, exactly as shops/resource.go:77 does.
func includesSkills(query url.Values) bool {
	for _, include := range query["include"] {
		for _, part := range strings.Split(include, ",") {
			if strings.TrimSpace(part) == "skills" {
				return true
			}
		}
	}
	return false
}

func handleGetJobsRequest(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			page, err := paginate.ParseParams(query, paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, err.Error())
				return
			}

			paged, err := NewStorage(d.Logger(), db).AllPagedProvider(d.Context())(page)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to retrieve jobs.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			items := ListFromAll(paged.Items)
			if includesSkills(query) {
				// One drain, never N per-id lookups: a 50-job page can reference
				// ~3,000 skills, and the registry cache only helps after the
				// first miss (design D4). When include is absent the skill
				// storage is not touched at all (NFR §8).
				all, err := skill.NewStorage(d.Logger(), db).DrainAllProvider(d.Context())()
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to retrieve skills for include=skills.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				byId := make(map[uint32]skill.RestModel, len(all))
				for _, s := range all {
					byId[s.Id] = s
				}
				items = WithResolvedSkills(items, byId)
			}

			envelope := paginate.EnvelopeFor(model.Paged[ListRestModel]{
				Items: items,
				Total: paged.Total,
				Page:  paged.Page,
			})
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]ListRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(items, envelope, r)
		}
	}
}
