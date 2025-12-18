package server

import (
	"encoding/json"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
)

// Marshal
// Deprecated : to be replaced with MarshalResponse
//
//goland:noinspection GoUnusedExportedFunction
func Marshal[A any](l logrus.FieldLogger) func(w http.ResponseWriter) func(si jsonapi.ServerInformation) func(slice A) {
	return func(w http.ResponseWriter) func(si jsonapi.ServerInformation) func(slice A) {
		return func(si jsonapi.ServerInformation) func(slice A) {
			return MarshalResponse[A](l)(w)(si)(make(map[string][]string))
		}
	}
}

//goland:noinspection GoUnusedExportedFunction
func MarshalResponse[A any](l logrus.FieldLogger) func(w http.ResponseWriter) func(si jsonapi.ServerInformation) func(queryParams map[string][]string) func(slice A) {
	return func(w http.ResponseWriter) func(si jsonapi.ServerInformation) func(queryParams map[string][]string) func(slice A) {
		return func(si jsonapi.ServerInformation) func(queryParams map[string][]string) func(slice A) {
			return func(queryParams map[string][]string) func(slice A) {
				return func(slice A) {
					d, err := jsonapi.MarshalToStruct(slice, si)
					if err != nil {
						l.WithError(err).Errorf("Unable to marshal models.")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					fd, errs := jsonapi.FilterSparseFields(d, queryParams)
					if errs != nil {
						ed, err := json.Marshal(errs[0])
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						_, err = w.Write(ed)
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					rd, err := json.Marshal(fd)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					_, err = w.Write(rd)
					if err != nil {
						l.WithError(err).Errorf("Unable to write response.")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
				}
			}
		}
	}
}
