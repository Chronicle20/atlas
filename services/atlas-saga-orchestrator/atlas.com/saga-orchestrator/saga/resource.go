package saga

import (
	"atlas-saga-orchestrator/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
)

// InitResource registers the transport routes with the router
func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(r *mux.Router, l logrus.FieldLogger) {
		r.HandleFunc("/sagas", rest.RegisterHandler(l)(si)("get_all_sagas", getAllSagasHandler)).Methods(http.MethodGet)
		r.HandleFunc("/sagas/{transactionId}", rest.RegisterHandler(l)(si)("get_saga_by_id", getSagaByIdHandler)).Methods(http.MethodGet)
	}
}

// getAllSagasHandler returns a handler for the GET /sagas endpoint
func getAllSagasHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get all sagas
		sms, err := NewProcessor(d.Logger(), d.Context()).GetAll()
		if err != nil {
			d.Logger().WithError(err).Error("Failed to retrieve sagas")
			w.WriteHeader(http.StatusInternalServerError)
		}

		rms, err := model.SliceMap(Transform)(model.FixedProvider(sms))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Error("Failed to retrieve sagas")
			w.WriteHeader(http.StatusInternalServerError)
		}

		// Marshal response
		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rms)
	}
}

// getSagaByIdHandler returns a handler for the GET /sagas/{transactionId} endpoint
func getSagaByIdHandler(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseTransactionId(d.Logger(), func(transactionId uuid.UUID) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Get the saga
			saga, err := NewProcessor(d.Logger(), d.Context()).GetById(transactionId)
			if err != nil {
				d.Logger().WithError(err).Error("Failed to retrieve sagas")
				w.WriteHeader(http.StatusInternalServerError)
			}

			rms, err := model.Map(Transform)(model.FixedProvider(saga))()
			if err != nil {
				d.Logger().WithError(err).Error("Failed to retrieve sagas")
				w.WriteHeader(http.StatusInternalServerError)
			}

			// Marshal response
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rms)
		}
	})
}
