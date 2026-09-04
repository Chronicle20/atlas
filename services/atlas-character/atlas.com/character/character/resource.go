package character

import (
	"atlas-character/rest"
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

// InitResource wires the /characters routes. nameReservedOf is injected
// (rather than imported) for the same reason teleport_rock's WorldIdOf is:
// pending_change already imports character for its apply path, so a direct
// import here would cycle. main.go closes the loop by passing
// pending_change.NameReservedFor(db); nil is a safe default (CheckNameValidity
// skips the reservation check entirely when nameReserved is nil).
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) func(nameReservedOf NameReservedFunc) server.RouteInitializer {
	return func(db *gorm.DB) func(nameReservedOf NameReservedFunc) server.RouteInitializer {
		return func(nameReservedOf NameReservedFunc) server.RouteInitializer {
			return func(router *mux.Router, l logrus.FieldLogger) {
				registerGet := rest.RegisterHandler(l)(si)
				r := router.PathPrefix("/characters").Subrouter()
				r.HandleFunc("", registerGet("get_characters_for_account_in_world", handleGetCharactersForAccountInWorld(db))).Methods(http.MethodGet).Queries("accountId", "{accountId}", "worldId", "{worldId}", "include", "{include}")
				r.HandleFunc("", registerGet("get_characters_for_account_in_world", handleGetCharactersForAccountInWorld(db))).Methods(http.MethodGet).Queries("accountId", "{accountId}", "worldId", "{worldId}")
				r.HandleFunc("", registerGet("get_characters_by_name", handleGetCharactersByName(db))).Methods(http.MethodGet).Queries("name", "{name}", "include", "{include}")
				r.HandleFunc("", registerGet("get_characters_by_name", handleGetCharactersByName(db))).Methods(http.MethodGet).Queries("name", "{name}")
				r.HandleFunc("", registerGet("get_characters", handleGetCharacters(db))).Methods(http.MethodGet)
				r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(si)("create_character", handleCreateCharacter(db))).Methods(http.MethodPost)
				r.HandleFunc("/name-validity", registerGet("get_name_validity", handleGetNameValidity(db, nameReservedOf))).Methods(http.MethodGet)
				r.HandleFunc("/{characterId}", registerGet("get_character", handleGetCharacter(db))).Methods(http.MethodGet).Queries("include", "{include}")
				r.HandleFunc("/{characterId}", registerGet("get_character", handleGetCharacter(db))).Methods(http.MethodGet)
				r.HandleFunc("/{characterId}", rest.RegisterInputHandler[RestModel](l)(si)("update_character", handleUpdateCharacter(db))).Methods(http.MethodPatch)
				r.HandleFunc("/{characterId}", rest.RegisterHandler(l)(si)("delete_character", handleDeleteCharacter(db))).Methods(http.MethodDelete)
				r.HandleFunc("/{characterId}/world-change", rest.RegisterInputHandler[WorldChangeInputRestModel](l)(si)("change_character_world", handleChangeCharacterWorld(db))).Methods(http.MethodPost)
			}
		}
	}
}

func handleGetCharacters(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, err.Error())
				return
			}

			paged, err := NewProcessor(d.Logger(), d.Context(), db).AllProvider(page, decoratorsFromInclude(r, d, c)...)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to get characters.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := model.SliceMap(Transform(d.Logger(), d.Context()))(model.FixedProvider(paged.Items))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
		}
	}
}

func handleGetCharactersForAccountInWorld(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
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

			page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, err.Error())
				return
			}

			paged, err := NewProcessor(d.Logger(), d.Context(), db).GetForAccountInWorldProvider(page, decoratorsFromInclude(r, d, c)...)(uint32(accountId), world.Id(worldId))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to get characters for account %d in world %d.", accountId, worldId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := model.SliceMap(Transform(d.Logger(), d.Context()))(model.FixedProvider(paged.Items))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
		}
	}
}

func decoratorsFromInclude(_ *http.Request, _ *rest.HandlerDependency, _ *rest.HandlerContext) []model.Decorator[Model] {
	decorators := make([]model.Decorator[Model], 0)
	return decorators
}

func handleGetCharactersByName(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			name, ok := mux.Vars(r)["name"]
			if !ok {
				d.Logger().Errorf("Unable to properly parse name from path.")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, err.Error())
				return
			}

			paged, err := NewProcessor(d.Logger(), d.Context(), db).GetForNameProvider(page, decoratorsFromInclude(r, d, c)...)(name)()
			if err != nil {
				d.Logger().WithError(err).Errorf("Getting character %s.", name)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := model.SliceMap(Transform(d.Logger(), d.Context()))(model.FixedProvider(paged.Items))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
		}
	}
}

func handleGetCharacter(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				cs, err := NewProcessor(d.Logger(), d.Context(), db).GetById(decoratorsFromInclude(r, d, c)...)(characterId)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				// Any non-404 failure (DB unavailable, decorator error, context
				// deadline) must surface as an error status. Falling through here
				// marshals the zero-value model and answers 200 with
				// {"id":"0", …all-zero attributes}, which callers decode without
				// error -- atlas-channel then runs the whole attack/damage
				// pipeline against character 0 with jobId 0, no skills and no
				// inventory instead of failing loudly.
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to retrieve character [%d].", characterId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				res, err := model.Map(Transform(d.Logger(), d.Context()))(model.FixedProvider(cs))()
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
	}
}

func handleCreateCharacter(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := Extract(input)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			cs, err := NewProcessor(d.Logger(), d.Context(), db).CreateAndEmit(uuid.New(), m, input.MapId)
			if err != nil {
				if errors.Is(err, errBlockedName) || errors.Is(err, errInvalidLevel) {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				d.Logger().WithError(err).Errorf("Creating character.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := model.Map(Transform(d.Logger(), d.Context()))(model.FixedProvider(cs))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	}
}

func handleDeleteCharacter(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := NewProcessor(d.Logger(), d.Context(), db).DeleteAndEmit(uuid.New(), characterId)
				if err != nil {
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}

// handleChangeCharacterWorld is the dedicated world-transfer route the saga
// (Task 13) calls. It exists separately from PATCH because world.Id is a
// byte and world 0 is a real, commonly-used world id: PATCH's "zero means
// absent" convention cannot express "transfer to world 0" (task-227
// controller ruling).
func handleChangeCharacterWorld(db *gorm.DB) rest.InputHandler[WorldChangeInputRestModel] {
	return func(d *rest.HandlerDependency, _ *rest.HandlerContext, input WorldChangeInputRestModel) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				transactionId := input.TransactionId
				if transactionId == uuid.Nil {
					transactionId = uuid.New()
				}
				err := NewProcessor(d.Logger(), d.Context(), db).ChangeWorldAndEmit(transactionId, characterId, input.NewWorldId)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					d.Logger().WithError(err).Errorf("Changing world for character [%d].", characterId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}

func handleUpdateCharacter(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, _ *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				transactionUuid := uuid.New()

				err := NewProcessor(d.Logger(), d.Context(), db).UpdateAndEmit(transactionUuid, characterId, input)
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
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				w.WriteHeader(http.StatusNoContent)
			}
		})
	}
}
