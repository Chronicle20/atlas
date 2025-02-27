package server

import (
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
)

//goland:noinspection GoUnusedExportedFunction
func Marshal[A any](l logrus.FieldLogger) func(w http.ResponseWriter) func(si jsonapi.ServerInformation) func(slice A) {
	return func(w http.ResponseWriter) func(si jsonapi.ServerInformation) func(slice A) {
		return func(si jsonapi.ServerInformation) func(slice A) {
			return func(slice A) {
				res, err := jsonapi.MarshalWithURLs(slice, si)
				if err != nil {
					l.WithError(err).Errorf("Unable to marshal models.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				_, err = w.Write(res)
				if err != nil {
					l.WithError(err).Errorf("Unable to write response.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
		}
	}
}
