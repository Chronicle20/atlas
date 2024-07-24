package monster

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
	getMonsters = "get_monsters"
	getMonster  = "get_monster"
)

func InitResource(si jsonapi.ServerInformation) func(router *mux.Router, l logrus.FieldLogger, db *gorm.DB) {
	return func(router *mux.Router, l logrus.FieldLogger, db *gorm.DB) {
		mRouter := router.PathPrefix("/monsters").Subrouter()
		mRouter.HandleFunc("", registerGetAllMonsters(si)(l, db)).Queries("filter[drops.item_id]", "{item_id}").Methods(http.MethodGet)
		mRouter.HandleFunc("", registerGetAllMonsters(si)(l, db)).Methods(http.MethodGet)
		mRouter.HandleFunc("/{monsterId}", registerGetMonster(si)(l, db)).Methods(http.MethodGet)
	}
}

type IdHandler func(monsterId uint32) http.HandlerFunc

func ParseId(l logrus.FieldLogger, next IdHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		monsterId, err := strconv.Atoi(mux.Vars(r)["monsterId"])
		if err != nil {
			l.Errorf("Unable to properly parse monsterId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint32(monsterId))(w, r)
	}
}

func registerGetAllMonsters(si jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
		return rest.RetrieveSpan(getMonsters, func(span opentracing.Span) http.HandlerFunc {
			return handleGetMonsters(si)(l, db)(span)
		})
	}
}

func handleGetMonsters(si jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) http.HandlerFunc {
		return func(span opentracing.Span) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				vars := mux.Vars(r)

				if val, ok := vars["item_id"]; ok {
					var itemId uint64
					itemId, err := strconv.ParseUint(val, 10, 32)
					if err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					ms := GetWhoDropItem(l, db)(uint32(itemId))

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
			}
		}
	}
}

func registerGetMonster(si jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
		return rest.RetrieveSpan(getMonster, func(span opentracing.Span) http.HandlerFunc {
			return ParseId(l, func(monsterId uint32) http.HandlerFunc {
				return handleGetMonster(si)(l, db)(span)(monsterId)
			})
		})
	}
}

func handleGetMonster(si jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) func(monsterId uint32) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) func(monsterId uint32) http.HandlerFunc {
		return func(span opentracing.Span) func(monsterId uint32) http.HandlerFunc {
			return func(monsterId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					m, err := GetById(l, db)(monsterId)
					if err != nil {
						l.WithError(err).Errorf("Unable to locate monster [%d].", monsterId)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					res, err := model.Map(model.FixedProvider(m), Transform)()
					if err != nil {
						l.WithError(err).Errorf("Creating REST model.")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					server.Marshal[RestModel](l)(w)(si)(res)
				}
			}
		}
	}
}
