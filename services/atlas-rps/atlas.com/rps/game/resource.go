package game

import (
	"atlas-rps/rest"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

// InitResource wires the REST surface for RPS sessions:
//   - POST /rps/games                 dispatches StartRPSGame (the entry
//     point atlas-saga-orchestrator calls to open a session at an NPC).
//   - GET  /rps/games/{characterId}   returns the character's active
//     session, or 404 when none exists.
//
// newProcessor supplies a fully-wired Processor (real ladder included) for
// each request. This package cannot build one itself: "atlas-rps/configuration"
// imports "atlas-rps/game" for the Ladder/Rung types, so game importing
// configuration back would be a cycle (see LadderProvider's doc). The
// composition root wiring main.go's server bootstrap supplies the concrete
// factory, mirroring how kafka/consumer/rps wires its own Processor.
func InitResource(si jsonapi.ServerInformation, newProcessor ProcessorFactory) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)
		registerInput := rest.RegisterInputHandler[RestModel](l)(si)

		r := router.PathPrefix("/rps/games").Subrouter()
		r.HandleFunc("", registerInput("create_rps_game", handleCreateGame(newProcessor))).Methods(http.MethodPost)
		r.HandleFunc("/{characterId}", registerGet("get_rps_game", handleGetGame(newProcessor))).Methods(http.MethodGet)
	}
}

// handleCreateGame handles POST /rps/games: it disposes any stale session
// for the character and opens a fresh rung-0 session, emitting GameOpened.
func handleCreateGame(newProcessor ProcessorFactory) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := newProcessor(d.Logger(), d.Context()).StartAndEmit(rm.CharacterId, rm.WorldId, rm.ChannelId, rm.NpcId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Starting RPS session for character [%d].", rm.CharacterId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := model.Map(Transform)(model.FixedProvider(m))()
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
}

// handleGetGame handles GET /rps/games/{characterId}: it returns the active
// session with its current-rung prize resolved from the real ladder, or 404
// when the character has no active session.
func handleGetGame(newProcessor ProcessorFactory) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, prize, prizeOk, err := newProcessor(d.Logger(), d.Context()).Get(characterId)
				if err != nil {
					if errors.Is(err, ErrSessionNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					d.Logger().WithError(err).Errorf("Retrieving RPS session for character [%d].", characterId)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				rm, err := TransformWithPrize(m, prize, prizeOk)
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}
