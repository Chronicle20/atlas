package extraction

import (
	"atlas-wz-extractor/extraction/job"
	"atlas-wz-extractor/rest"
	"errors"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
)

func handleJobStatus(store job.Store) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			jobId := vars["jobId"]
			j, units, err := store.Get(d.Context(), jobId)
			if errors.Is(err, job.ErrNotFound) {
				http.Error(w, "job not found", http.StatusNotFound)
				return
			}
			if err != nil {
				d.Logger().WithError(err).Error("Get job failed")
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			rm := TransformJob(j, units)
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[JobRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
		}
	}
}
