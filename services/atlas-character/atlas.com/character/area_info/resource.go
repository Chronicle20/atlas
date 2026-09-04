package area_info

import (
	"atlas-character/rest"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			r := router.PathPrefix("/characters/{characterId}/area-info").Subrouter()
			r.HandleFunc("", registerGet("get_area_infos", handleGetAreaInfos)).Methods(http.MethodGet)
			r.HandleFunc("/{area}", registerGet("get_area_info", handleGetAreaInfo)).Methods(http.MethodGet)
			r.HandleFunc("/{area}", rest.RegisterInputHandler[PutRestModel](l)(db)(si)("put_area_info", handlePutAreaInfo)).Methods(http.MethodPut)
		}
	}
}

func parseArea(l logrus.FieldLogger, next func(area uint16) http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		areaStr, ok := mux.Vars(r)["area"]
		if !ok || areaStr == "" {
			l.Errorf("Unable to properly parse area from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		area, err := strconv.ParseUint(areaStr, 10, 16)
		if err != nil {
			l.WithError(err).Errorf("Unable to properly parse area from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(uint16(area))(w, r)
	}
}

func handleGetAreaInfos(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ms, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetAll(characterId)
			if err != nil {
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := TransformSlice(ms)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}

func handleGetAreaInfo(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return parseArea(d.Logger(), func(area uint16) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetByArea(characterId, area)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				res, err := model.Map(Transform)(model.FixedProvider(m))()
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
	})
}

func handlePutAreaInfo(d *rest.HandlerDependency, c *rest.HandlerContext, input PutRestModel) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return parseArea(d.Logger(), func(area uint16) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := NewBuilder().
					SetCharacterId(characterId).
					SetArea(area).
					SetInfo(input.Info).
					Build()
				if err != nil {
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				result, err := NewProcessor(d.Logger(), d.Context(), d.DB()).Put(m)
				if err != nil {
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				res, err := model.Map(Transform)(model.FixedProvider(result))()
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
	})
}
