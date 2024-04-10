package monster

import (
	"atlas-dis/rest"
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
				var res []byte
				var err error

				if val, ok := vars["item_id"]; ok {
					var itemId uint64
					itemId, err = strconv.ParseUint(val, 10, 32)
					if err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					res, err = jsonapi.MarshalWithURLs(TransformAll(GetWhoDropItem(l, db)(uint32(itemId))), si)
					if err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
				} else {
					res, err = jsonapi.MarshalWithURLs(TransformAll(GetAll(l, db)), si)
					if err != nil {
						l.WithError(err).Errorf("Unable to marshal models.")
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
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

					res, err := jsonapi.MarshalWithURLs(Transform(m), si)
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
}
