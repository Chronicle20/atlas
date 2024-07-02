package character

import (
	"atlas-character/rest"
	"atlas-character/tenant"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"io"
	"net/http"
	"strconv"
)

const (
	GetCharactersForAccountInWorld = "get_characters_for_account_in_world"
	GetCharactersByMap             = "get_characters_by_map"
	GetCharactersByName            = "get_characters_by_name"
	GetCharacter                   = "get_character"
	CreateCharacter                = "create_character"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := registerGetHandler(l)(db)(si)
			r := router.PathPrefix("/characters").Subrouter()
			r.HandleFunc("", registerGet(GetCharactersForAccountInWorld, handleGetCharactersForAccountInWorld)).Methods(http.MethodGet).Queries("accountId", "{accountId}", "worldId", "{worldId}")
			r.HandleFunc("", registerGet(GetCharactersByMap, handleGetCharactersByMap)).Methods(http.MethodGet).Queries("worldId", "{worldId}", "mapId", "{mapId}")
			r.HandleFunc("", registerGet(GetCharactersByName, handleGetCharactersByName)).Methods(http.MethodGet).Queries("name", "{name}")
			r.HandleFunc("", registerCreateHandler[RestModel](l)(db)(si)(CreateCharacter, handleCreateCharacter)).Methods(http.MethodPost)
			r.HandleFunc("/{characterId}", registerGet(GetCharacter, handleGetCharacter)).Methods(http.MethodGet)
		}
	}
}

type handlerDependency struct {
	l    logrus.FieldLogger
	db   *gorm.DB
	span opentracing.Span
}

type handlerContext struct {
	si jsonapi.ServerInformation
	t  tenant.Model
}

type getHandler func(d *handlerDependency, c *handlerContext) http.HandlerFunc

type createHandler[M any] func(d *handlerDependency, c *handlerContext, model M) http.HandlerFunc

func parseInput[M any](d *handlerDependency, c *handlerContext, next createHandler[M]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var model M

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		err = jsonapi.Unmarshal(body, &model)
		if err != nil {
			d.l.WithError(err).Errorln("Deserializing input", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next(d, c, model)(w, r)
	}
}

func registerGetHandler(l logrus.FieldLogger) func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler getHandler) http.HandlerFunc {
	return func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler getHandler) http.HandlerFunc {
		return func(si jsonapi.ServerInformation) func(handlerName string, handler getHandler) http.HandlerFunc {
			return func(handlerName string, handler getHandler) http.HandlerFunc {
				return rest.RetrieveSpan(handlerName, func(span opentracing.Span) http.HandlerFunc {
					fl := l.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
					return rest.ParseTenant(fl, func(tenant tenant.Model) http.HandlerFunc {
						return handler(&handlerDependency{l: fl, db: db, span: span}, &handlerContext{si: si, t: tenant})
					})
				})
			}
		}
	}
}

func registerCreateHandler[M any](l logrus.FieldLogger) func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler createHandler[M]) http.HandlerFunc {
	return func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler createHandler[M]) http.HandlerFunc {
		return func(si jsonapi.ServerInformation) func(handlerName string, handler createHandler[M]) http.HandlerFunc {
			return func(handlerName string, handler createHandler[M]) http.HandlerFunc {
				return rest.RetrieveSpan(handlerName, func(span opentracing.Span) http.HandlerFunc {
					fl := l.WithFields(logrus.Fields{"originator": handlerName, "type": "rest_handler"})
					return rest.ParseTenant(fl, func(tenant tenant.Model) http.HandlerFunc {
						d := &handlerDependency{l: fl, db: db, span: span}
						c := &handlerContext{si: si, t: tenant}
						return parseInput[M](d, c, handler)
					})
				})
			}
		}
	}
}

func handleGetCharactersForAccountInWorld(d *handlerDependency, c *handlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accountId, err := strconv.Atoi(mux.Vars(r)["accountId"])
		if err != nil {
			d.l.WithError(err).Errorf("Unable to properly parse accountId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		worldId, err := strconv.Atoi(mux.Vars(r)["worldId"])
		if err != nil {
			d.l.WithError(err).Errorf("Unable to properly parse worldId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		cs, err := GetForAccountInWorld(d.l, d.db, c.t)(uint32(accountId), byte(worldId), InventoryModelDecorator(d.l, d.db, c.t))
		if err != nil {
			d.l.WithError(err).Errorf("Unable to get characters for account %d in world %d.", accountId, worldId)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.TransformAll(cs, Transform)
		if err != nil {
			d.l.WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[[]RestModel](d.l)(w)(c.si)(res)
	}
}

func handleGetCharactersByMap(d *handlerDependency, c *handlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		worldId, err := strconv.Atoi(mux.Vars(r)["worldId"])
		if err != nil {
			d.l.WithError(err).Errorf("Unable to properly parse worldId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mapId, err := strconv.Atoi(mux.Vars(r)["mapId"])
		if err != nil {
			d.l.WithError(err).Errorf("Unable to properly parse mapId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		cs, err := GetForMapInWorld(d.l, d.db, c.t)(byte(worldId), uint32(mapId), InventoryModelDecorator(d.l, d.db, c.t))
		if err != nil {
			d.l.WithError(err).Errorf("Unable to get characters for map %d in world %d.", mapId, worldId)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.TransformAll(cs, Transform)
		if err != nil {
			d.l.WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[[]RestModel](d.l)(w)(c.si)(res)
	}
}

func handleGetCharactersByName(d *handlerDependency, c *handlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name, ok := mux.Vars(r)["name"]
		if !ok {
			d.l.Errorf("Unable to properly parse name from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		cs, err := GetForName(d.l, d.db, c.t)(name, InventoryModelDecorator(d.l, d.db, c.t))
		if err != nil {
			d.l.WithError(err).Errorf("Getting character %s.", name)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.TransformAll(cs, Transform)
		if err != nil {
			d.l.WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[[]RestModel](d.l)(w)(c.si)(res)
	}
}

func handleGetCharacter(d *handlerDependency, c *handlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		characterId, err := strconv.Atoi(mux.Vars(r)["characterId"])
		if err != nil {
			d.l.WithError(err).Errorf("Unable to properly parse characterId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		cs, err := GetById(d.l, d.db, c.t)(uint32(characterId), InventoryModelDecorator(d.l, d.db, c.t))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if err != nil {
			d.l.WithError(err).Errorf("Getting character %d.", characterId)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.Transform(cs, Transform)
		if err != nil {
			d.l.WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[RestModel](d.l)(w)(c.si)(res)
	}
}

func handleCreateCharacter(d *handlerDependency, c *handlerContext, input RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m, err := Extract(input)
		if err != nil {
			d.l.WithError(err).Errorf("Creating model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		cs, err := Create(d.l, d.db, d.span, c.t)(m)
		if err != nil {
			if errors.Is(err, blockedNameErr) || errors.Is(err, invalidLevelErr) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			d.l.WithError(err).Errorf("Creating character.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.Transform(cs, Transform)
		if err != nil {
			d.l.WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[RestModel](d.l)(w)(c.si)(res)
	}
}
