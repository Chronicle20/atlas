package character

import (
	"atlas-character/rest"
	"atlas-character/tenant"
	"errors"
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
	GetCharactersForAccountInWorld = "get_characters_for_account_in_world"
	GetCharactersByMap             = "get_characters_by_map"
	GetCharactersByName            = "get_characters_by_name"
	GetCharacter                   = "get_character"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			register := registerHandler(l)(db)(si)
			r := router.PathPrefix("/characters").Subrouter()
			r.HandleFunc("", register(GetCharactersForAccountInWorld, handleGetCharactersForAccountInWorld)).Methods(http.MethodGet).Queries("accountId", "{accountId}", "worldId", "{worldId}")
			r.HandleFunc("", register(GetCharactersByMap, handleGetCharactersByMap)).Methods(http.MethodGet).Queries("worldId", "{worldId}", "mapId", "{mapId}")
			r.HandleFunc("", register(GetCharactersByName, handleGetCharactersByName)).Methods(http.MethodGet).Queries("name", "{name}")
			r.HandleFunc("/{characterId}", register(GetCharacter, handleGetCharacter)).Methods(http.MethodGet)
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

type handler func(d *handlerDependency, c *handlerContext) http.HandlerFunc

func registerHandler(l logrus.FieldLogger) func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler handler) http.HandlerFunc {
	return func(db *gorm.DB) func(si jsonapi.ServerInformation) func(handlerName string, handler handler) http.HandlerFunc {
		return func(si jsonapi.ServerInformation) func(handlerName string, handler handler) http.HandlerFunc {
			return func(handlerName string, handler handler) http.HandlerFunc {
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

		server.Marshal[[]RestModel](d.l)(w)(c.si)(TransformAll(cs))
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

		cs, err := GetForMapInWorld(d.l, d.db, c.t)(byte(worldId), uint32(mapId))
		if err != nil {
			d.l.WithError(err).Errorf("Unable to get characters for map %d in world %d.", mapId, worldId)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[[]RestModel](d.l)(w)(c.si)(TransformAll(cs))
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

		cs, err := GetForName(d.l, d.db, c.t)(name)
		if err != nil {
			d.l.WithError(err).Errorf("Getting character %s.", name)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[[]RestModel](d.l)(w)(c.si)(TransformAll(cs))
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
		chr, err := GetById(d.l, d.db, c.t)(uint32(characterId))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if err != nil {
			d.l.WithError(err).Errorf("Getting character %d.", characterId)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[RestModel](d.l)(w)(c.si)(Transform(chr))
	}
}
