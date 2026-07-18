package job

import (
	"atlas-data/rest"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)

		r := router.PathPrefix("/data/jobs").Subrouter()
		r.HandleFunc("/{jobId}/skills",
			registerGet("get_job_skills", handleGetJobSkills())).Methods(http.MethodGet)
	}
}

func handleGetJobSkills() func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseJobId(d.Logger(), func(jobId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, ok := NewProcessor(d.Logger(), d.Context()).GetSkillsForJob(jobId)
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
