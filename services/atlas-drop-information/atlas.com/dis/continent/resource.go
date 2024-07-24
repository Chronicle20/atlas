package continent

import (
	"atlas-dis/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

const (
	getContinents = "get_continents"
)

func InitResource(si jsonapi.ServerInformation) func(router *mux.Router, l logrus.FieldLogger, db *gorm.DB) {
	return func(router *mux.Router, l logrus.FieldLogger, db *gorm.DB) {
		mRouter := router.PathPrefix("/continents").Subrouter()
		mRouter.HandleFunc("", registerGetAllContinents(si)(l, db)).Methods(http.MethodGet)
	}
}

func registerGetAllContinents(si jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
		return rest.RetrieveSpan(getContinents, func(span opentracing.Span) http.HandlerFunc {
			return handleGetContinents(si)(l, db)(span)
		})
	}
}

func handleGetContinents(si jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) http.HandlerFunc {
		return func(span opentracing.Span) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				cs := GetAll(l, db)
				res, err := model.SliceMap(model.FixedProvider(cs), Transform)()
				if err != nil {
					l.WithError(err).Errorf("Unable to marshal models.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				server.Marshal[[]RestModel](l)(w)(si)(res)
			}
		}
	}
}
