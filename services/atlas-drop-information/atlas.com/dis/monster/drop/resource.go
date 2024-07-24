package drop

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
	"strconv"
)

const (
	getAllDrops = "get_all_drops"
)

func InitResource(si jsonapi.ServerInformation) func(router *mux.Router, l logrus.FieldLogger, db *gorm.DB) {
	return func(router *mux.Router, l logrus.FieldLogger, db *gorm.DB) {
		dRouter := router.PathPrefix("/drops").Subrouter()
		dRouter.HandleFunc("", registerDrops(si)(l, db)).Queries("monster_id", "{monster_id}").Methods(http.MethodGet)
		dRouter.HandleFunc("", registerDrops(si)(l, db)).Methods(http.MethodGet)
	}
}

func registerDrops(si jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
		return rest.RetrieveSpan(getAllDrops, func(span opentracing.Span) http.HandlerFunc {
			return handleGetAllDrops(si)(l, db)(span)
		})
	}
}

func handleGetAllDrops(si jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) http.HandlerFunc {
		return func(span opentracing.Span) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				vars := mux.Vars(r)
				if val, ok := vars["monster_id"]; ok {
					var monsterId uint64
					monsterId, err := strconv.ParseUint(val, 10, 32)
					if err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					ms := GetForMonster(l, db)(uint32(monsterId))
					res, err := model.SliceMap(model.FixedProvider(ms), Transform)()
					if err != nil {
						l.WithError(err).Errorf("Creating REST model.")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					server.Marshal[[]RestModel](l)(w)(si)(res)
					return

				}

				ms := GetAll(l, db)
				res, err := model.SliceMap(model.FixedProvider(ms), Transform)()
				if err != nil {
					l.WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				server.Marshal[[]RestModel](l)(w)(si)(res)
				return
			}
		}
	}
}
