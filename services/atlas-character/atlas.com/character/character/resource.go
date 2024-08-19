package character

import (
	"atlas-character/rest"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"strings"
)

const (
	GetCharactersForAccountInWorld = "get_characters_for_account_in_world"
	GetCharactersByMap             = "get_characters_by_map"
	GetCharactersByName            = "get_characters_by_name"
	GetCharacter                   = "get_character"
	DeleteCharacter                = "delete_character"
	CreateCharacter                = "create_character"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			r := router.PathPrefix("/characters").Subrouter()
			r.HandleFunc("", registerGet(GetCharactersForAccountInWorld, handleGetCharactersForAccountInWorld)).Methods(http.MethodGet).Queries("accountId", "{accountId}", "worldId", "{worldId}", "include", "{include}")
			r.HandleFunc("", registerGet(GetCharactersForAccountInWorld, handleGetCharactersForAccountInWorld)).Methods(http.MethodGet).Queries("accountId", "{accountId}", "worldId", "{worldId}")
			r.HandleFunc("", registerGet(GetCharactersByMap, handleGetCharactersByMap)).Methods(http.MethodGet).Queries("worldId", "{worldId}", "mapId", "{mapId}", "include", "{include}")
			r.HandleFunc("", registerGet(GetCharactersByMap, handleGetCharactersByMap)).Methods(http.MethodGet).Queries("worldId", "{worldId}", "mapId", "{mapId}")
			r.HandleFunc("", registerGet(GetCharactersByName, handleGetCharactersByName)).Methods(http.MethodGet).Queries("name", "{name}", "include", "{include}")
			r.HandleFunc("", registerGet(GetCharactersByName, handleGetCharactersByName)).Methods(http.MethodGet).Queries("name", "{name}")
			r.HandleFunc("", rest.RegisterCreateHandler[RestModel](l)(db)(si)(CreateCharacter, handleCreateCharacter)).Methods(http.MethodPost)
			r.HandleFunc("/{characterId}", registerGet(GetCharacter, handleGetCharacter)).Methods(http.MethodGet).Queries("include", "{include}")
			r.HandleFunc("/{characterId}", registerGet(GetCharacter, handleGetCharacter)).Methods(http.MethodGet)
			r.HandleFunc("/{characterId}", rest.RegisterHandler(l)(db)(si)(DeleteCharacter, handleDeleteCharacter)).Methods(http.MethodDelete)
		}
	}
}

func handleGetCharactersForAccountInWorld(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accountId, err := strconv.Atoi(mux.Vars(r)["accountId"])
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to properly parse accountId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		worldId, err := strconv.Atoi(mux.Vars(r)["worldId"])
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to properly parse worldId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		cs, err := GetForAccountInWorld(d.Logger(), d.DB(), c.Tenant())(uint32(accountId), byte(worldId), decoratorsFromInclude(r, d, c)...)
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to get characters for account %d in world %d.", accountId, worldId)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.SliceMap(model.FixedProvider(cs), Transform)()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[[]RestModel](d.Logger())(w)(c.ServerInformation())(res)
	}
}

func decoratorsFromInclude(r *http.Request, d *rest.HandlerDependency, c *rest.HandlerContext) []model.Decorator[Model] {
	var decorators = make([]model.Decorator[Model], 0)
	include := mux.Vars(r)["include"]
	if strings.Contains(include, "inventory") {
		decorators = append(decorators, InventoryModelDecorator(d.Logger(), d.DB(), d.Span(), c.Tenant()))
	}
	return decorators
}

func handleGetCharactersByMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		worldId, err := strconv.Atoi(mux.Vars(r)["worldId"])
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to properly parse worldId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mapId, err := strconv.Atoi(mux.Vars(r)["mapId"])
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to properly parse mapId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		cs, err := GetForMapInWorld(d.Logger(), d.DB(), c.Tenant())(byte(worldId), uint32(mapId), decoratorsFromInclude(r, d, c)...)
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to get characters for map %d in world %d.", mapId, worldId)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.SliceMap(model.FixedProvider(cs), Transform)()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[[]RestModel](d.Logger())(w)(c.ServerInformation())(res)
	}
}

func handleGetCharactersByName(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name, ok := mux.Vars(r)["name"]
		if !ok {
			d.Logger().Errorf("Unable to properly parse name from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		cs, err := GetForName(d.Logger(), d.DB(), c.Tenant())(name, decoratorsFromInclude(r, d, c)...)
		if err != nil {
			d.Logger().WithError(err).Errorf("Getting character %s.", name)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.SliceMap(model.FixedProvider(cs), Transform)()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[[]RestModel](d.Logger())(w)(c.ServerInformation())(res)
	}
}

func handleGetCharacter(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cs, err := GetById(d.Logger(), d.DB(), c.Tenant())(characterId, decoratorsFromInclude(r, d, c)...)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			if err != nil {
				d.Logger().WithError(err).Errorf("Getting character %d.", characterId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := model.Map(model.FixedProvider(cs), Transform)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}

func handleCreateCharacter(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m, err := Extract(input)
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		cs, err := Create(d.Logger(), d.DB(), d.Span(), c.Tenant())(m)
		if err != nil {
			if errors.Is(err, blockedNameErr) || errors.Is(err, invalidLevelErr) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			d.Logger().WithError(err).Errorf("Creating character.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.Map(model.FixedProvider(cs), Transform)()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
	}
}

func handleDeleteCharacter(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := Delete(d.Logger(), d.DB(), d.Span(), c.Tenant())(characterId)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	})
}
