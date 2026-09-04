package report

import (
	"atlas-ban/rest"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			register := rest.RegisterHandler(l)(si)
			registerInput := rest.RegisterInputHandler[RestModel](l)(si)

			r := router.PathPrefix("/reports").Subrouter()
			r.HandleFunc("/", register("get_reports", handleGetReports(db))).Methods(http.MethodGet)
			r.HandleFunc("/{reportId}", register("get_report", handleGetReportById(db))).Methods(http.MethodGet)
			r.HandleFunc("/{reportId}", registerInput("update_report_status", handleUpdateReportStatus(db))).Methods(http.MethodPatch)
		}
	}
}

func handleGetReports(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			statusStr := r.URL.Query().Get("status")

			var reports []Model
			var err error
			if statusStr != "" {
				s := Status(statusStr)
				if !s.Valid() {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				reports, err = NewProcessor(d.Logger(), d.Context(), db).GetByStatus(s)
			} else {
				reports, err = NewProcessor(d.Logger(), d.Context(), db).GetByTenant()
			}
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to locate reports.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := TransformSlice(reports)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	}
}

func handleGetReportById(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseReportId(d.Logger(), func(reportId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := NewProcessor(d.Logger(), d.Context(), db).GetById(reportId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to retrieve report [%s].", reportId)
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				res, err := Transform(m)
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}

func handleUpdateReportStatus(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return rest.ParseReportId(d.Logger(), func(reportId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				if input.Id != uuid.Nil && input.Id != reportId {
					d.Logger().Errorln("Report ID does not match URL")
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				m, err := NewProcessor(d.Logger(), d.Context(), db).UpdateStatus(reportId, Status(input.Status))
				if err != nil {
					if errors.Is(err, ErrInvalidStatus) {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					d.Logger().WithError(err).Errorf("Unable to update report [%s].", reportId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				res, err := Transform(m)
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}
