package drop

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
	getMonsterDrops = "get_monster_drops"
)

func InitResource(si jsonapi.ServerInformation) func(router *mux.Router, l logrus.FieldLogger, db *gorm.DB) {
	return func(router *mux.Router, l logrus.FieldLogger, db *gorm.DB) {
		eRouter := router.PathPrefix("/monsters").Subrouter()
		eRouter.HandleFunc("/{monsterId}/drops", registerMonsterDrops(si)(l, db)).Methods(http.MethodGet)
	}
}

type MonsterIdHandler func(monsterId uint32) http.HandlerFunc

func ParseMonsterId(l logrus.FieldLogger, next MonsterIdHandler) http.HandlerFunc {
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

func registerMonsterDrops(si jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
		return rest.RetrieveSpan(getMonsterDrops, func(span opentracing.Span) http.HandlerFunc {
			return ParseMonsterId(l, func(monsterId uint32) http.HandlerFunc {
				return handleMonsterDrops(si)(l, db)(span)(monsterId)
			})
		})
	}
}

func handleMonsterDrops(si jsonapi.ServerInformation) func(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) func(monsterId uint32) http.HandlerFunc {
	return func(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) func(monsterId uint32) http.HandlerFunc {
		return func(span opentracing.Span) func(monsterId uint32) http.HandlerFunc {
			return func(monsterId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					res, err := jsonapi.MarshalWithURLs(TransformAll(GetForMonster(l, db)(monsterId)), si)
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
