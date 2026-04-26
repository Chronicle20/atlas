package server

import (
	"encoding/json"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

// MarshalPaginatedResponse marshals a slice into a JSON:API document with
// meta and links populated from the supplied paginate.Envelope. It mirrors
// the existing MarshalResponse plumbing (sparse-fields filtering, error
// handling) but injects the pagination envelope before write.
//
//goland:noinspection GoUnusedExportedFunction
func MarshalPaginatedResponse[A any](l logrus.FieldLogger) func(w http.ResponseWriter) func(si jsonapi.ServerInformation) func(queryParams map[string][]string) func(slice A, env paginate.Envelope, req *http.Request) {
	return func(w http.ResponseWriter) func(si jsonapi.ServerInformation) func(queryParams map[string][]string) func(slice A, env paginate.Envelope, req *http.Request) {
		return func(si jsonapi.ServerInformation) func(queryParams map[string][]string) func(slice A, env paginate.Envelope, req *http.Request) {
			return func(queryParams map[string][]string) func(slice A, env paginate.Envelope, req *http.Request) {
				return func(slice A, env paginate.Envelope, req *http.Request) {
					d, err := jsonapi.MarshalToStruct(slice, si)
					if err != nil {
						l.WithError(err).Errorf("Unable to marshal models.")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					d.Meta = env.Meta()
					d.Links = env.BuildLinks(req)

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
					if _, err = w.Write(rd); err != nil {
						l.WithError(err).Errorf("Unable to write response.")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
				}
			}
		}
	}
}
