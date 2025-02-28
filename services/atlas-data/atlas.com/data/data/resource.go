package data

import (
	"atlas-data/rest"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		r := router.PathPrefix("/data").Subrouter()
		r.HandleFunc("", rest.RegisterHandler(l)(si)("register_data", handleRegisterData)).Methods(http.MethodPatch)
	}
}

func handleRegisterData(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := RegisterData(d.Logger())(d.Context())
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to process data registration request.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}
