package character

import (
	"atlas-character/rest"
	"errors"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
	"strconv"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			r := router.PathPrefix("/characters").Subrouter()
			r.HandleFunc("", registerGet("get_characters_for_account_in_world", handleGetCharactersForAccountInWorld)).Methods(http.MethodGet).Queries("accountId", "{accountId}", "worldId", "{worldId}", "include", "{include}")
			r.HandleFunc("", registerGet("get_characters_for_account_in_world", handleGetCharactersForAccountInWorld)).Methods(http.MethodGet).Queries("accountId", "{accountId}", "worldId", "{worldId}")
			r.HandleFunc("", registerGet("get_characters_by_map", handleGetCharactersByMap)).Methods(http.MethodGet).Queries("worldId", "{worldId}", "mapId", "{mapId}", "include", "{include}")
			r.HandleFunc("", registerGet("get_characters_by_map", handleGetCharactersByMap)).Methods(http.MethodGet).Queries("worldId", "{worldId}", "mapId", "{mapId}")
			r.HandleFunc("", registerGet("get_characters_by_name", handleGetCharactersByName)).Methods(http.MethodGet).Queries("name", "{name}", "include", "{include}")
			r.HandleFunc("", registerGet("get_characters_by_name", handleGetCharactersByName)).Methods(http.MethodGet).Queries("name", "{name}")
			r.HandleFunc("", registerGet("get_characters", handleGetCharacters)).Methods(http.MethodGet)
			r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(db)(si)("create_character", handleCreateCharacter)).Methods(http.MethodPost)
			r.HandleFunc("/{characterId}", registerGet("get_character", handleGetCharacter)).Methods(http.MethodGet).Queries("include", "{include}")
			r.HandleFunc("/{characterId}", registerGet("get_character", handleGetCharacter)).Methods(http.MethodGet)
			r.HandleFunc("/{characterId}", rest.RegisterInputHandler[RestModel](l)(db)(si)("update_character", handleUpdateCharacter)).Methods(http.MethodPatch)
			r.HandleFunc("/{characterId}", rest.RegisterHandler(l)(db)(si)("delete_character", handleDeleteCharacter)).Methods(http.MethodDelete)
		}
	}
}

func handleGetCharacters(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cs, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetAll(decoratorsFromInclude(r, d, c)...)
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to get characters.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.SliceMap(Transform)(model.FixedProvider(cs))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
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

		cs, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetForAccountInWorld(decoratorsFromInclude(r, d, c)...)(uint32(accountId), world.Id(worldId))
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to get characters for account %d in world %d.", accountId, worldId)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.SliceMap(Transform)(model.FixedProvider(cs))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
	}
}

func decoratorsFromInclude(r *http.Request, d *rest.HandlerDependency, _ *rest.HandlerContext) []model.Decorator[Model] {
	var decorators = make([]model.Decorator[Model], 0)
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

		cs, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetForMapInWorld(decoratorsFromInclude(r, d, c)...)(world.Id(worldId), _map.Id(mapId))
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to get characters for map %d in world %d.", mapId, worldId)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.SliceMap(Transform)(model.FixedProvider(cs))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
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

		cs, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetForName(decoratorsFromInclude(r, d, c)...)(name)
		if err != nil {
			d.Logger().WithError(err).Errorf("Getting character %s.", name)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.SliceMap(Transform)(model.FixedProvider(cs))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
	}
}

func handleGetCharacter(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			cs, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetById(decoratorsFromInclude(r, d, c)...)(characterId)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			res, err := model.Map(Transform)(model.FixedProvider(cs))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
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
		cs, err := NewProcessor(d.Logger(), d.Context(), d.DB()).CreateAndEmit(uuid.New(), m)
		if err != nil {
			if errors.Is(err, blockedNameErr) || errors.Is(err, invalidLevelErr) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			d.Logger().WithError(err).Errorf("Creating character.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.Map(Transform)(model.FixedProvider(cs))()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
	}
}

func handleDeleteCharacter(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := NewProcessor(d.Logger(), d.Context(), d.DB()).DeleteAndEmit(uuid.New(), characterId)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	})
}

func handleUpdateCharacter(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			transactionUuid := uuid.New()

			err := NewProcessor(d.Logger(), d.Context(), d.DB()).UpdateAndEmit(transactionUuid, characterId, input)
			if err != nil {
				if err.Error() == "invalid or duplicate name" ||
					err.Error() == "invalid hair ID" ||
					err.Error() == "invalid face ID" ||
					err.Error() == "invalid gender value" ||
					err.Error() == "invalid skin color value" ||
					err.Error() == "invalid GM value" ||
					err.Error() == "invalid map ID or character cannot access this map" {
					d.Logger().WithError(err).Errorf("Validation error updating character [%d].", characterId)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				d.Logger().WithError(err).Errorf("Error updating character [%d].", characterId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		}
	})
}
