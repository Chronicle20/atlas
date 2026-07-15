package holding

import (
	"atlas-mts/rest"
	"errors"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// InitResource registers the holding routes:
//   - GET  /characters/{characterId}/mts/holding                          — a character's take-home holdings
//   - POST /characters/{characterId}/mts/holding/{holdingId}/take-home    — initiate a take-home (WithdrawFromMts saga)
//
// On GET, an optional ?worldId= query param narrows the result to a single world;
// absent, all of the character's holdings are returned. The POST initiates the
// WithdrawFromMts saga (release custody + grant to inventory); it does NOT
// soft-delete the holding row directly — the saga's ReleaseFromMtsHolding custody
// command does that (idempotently on replay).
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			registerInput := rest.RegisterInputHandler[TakeHomeRestModel](l)(db)(si)

			r := router.PathPrefix("/characters/{characterId}/mts/holding").Subrouter()
			r.HandleFunc("", registerGet("get_character_holdings", handleGetCharacterHoldings)).Methods(http.MethodGet)
			r.HandleFunc("/{holdingId}/take-home", registerInput("take_home_holding", handleTakeHome)).Methods(http.MethodPost)
		}
	}
}

// handleTakeHome initiates the owner's take-home of a holding into inventory by
// emitting a WithdrawFromMts saga. The response is 202 Accepted carrying the
// allocated transaction id — the holding is released and the item granted only
// when the custody saga lands (ReleaseFromMtsHolding + AcceptToCharacter).
//
// Owner-only check (mirroring the listing seller-only check, Task 4.2): the
// holding is loaded by id, and the take-home proceeds only when the requesting
// characterId (path var) is the holding's owner; otherwise 403.
func handleTakeHome(d *rest.HandlerDependency, c *rest.HandlerContext, rm TakeHomeRestModel) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return rest.ParseHoldingId(d.Logger(), func(holdingId string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				p := NewProcessor(d.Logger(), d.Context(), d.DB())

				// Load the holding for the owner-only check.
				m, err := p.GetById(holdingId)
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					d.Logger().WithError(err).Errorf("Retrieving holding [%s] for take-home.", holdingId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				// Owner-only: only the holding's owner may take it home.
				if m.OwnerId() != characterId {
					d.Logger().Errorf("Character [%d] attempted to take home holding [%s] owned by [%d]; forbidden.", characterId, holdingId, m.OwnerId())
					w.WriteHeader(http.StatusForbidden)
					return
				}

				txnId, err := p.TakeHome(holdingId, characterId, m.WorldId(), rm.InventoryType, rm.Slot)
				if err != nil {
					d.Logger().WithError(err).Errorf("Initiating take-home of holding [%s] for character [%d].", holdingId, characterId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				res := TakeHomeRestModel{
					Id:            txnId.String(),
					InventoryType: rm.InventoryType,
					Slot:          rm.Slot,
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				w.WriteHeader(http.StatusAccepted)
				server.MarshalResponse[TakeHomeRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	})
}

// handleGetCharacterHoldings is a game-capped list (take-home items are bounded
// by however many the character has taken out of custody but not yet withdrawn
// to inventory) — page[size] defaults to and caps at paginate.MaxPageSize
// (task-117).
func handleGetCharacterHoldings(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, perr := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
			if perr != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			p := NewProcessor(d.Logger(), d.Context(), d.DB())

			var paged model.Paged[Model]
			var err error
			if v := r.URL.Query().Get("worldId"); v != "" {
				worldId, werr := strconv.ParseUint(v, 10, 8)
				if werr != nil {
					d.Logger().WithError(werr).Errorf("Unable to parse worldId query for character [%d].", characterId)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				paged, err = p.ByOwnerPagedProvider(world.Id(byte(worldId)), characterId, page)()
			} else {
				paged, err = p.ByCharacterPagedProvider(characterId, page)()
			}
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving holdings for character [%d].", characterId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
		}
	})
}
