package pending_change

import (
	"atlas-character/rest"
	"encoding/json"
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
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			r := router.PathPrefix("/characters/{characterId}/pending-changes").Subrouter()
			r.HandleFunc("", registerGet("get_pending_changes", handleGetPendingChanges)).Methods(http.MethodGet)
			r.HandleFunc("", rest.RegisterInputHandler[CreateInputRestModel](l)(db)(si)("create_pending_change", handleCreatePendingChange)).Methods(http.MethodPost)
			r.HandleFunc("/{id}", registerGet("cancel_pending_change", handleCancelPendingChange)).Methods(http.MethodDelete)
			r.HandleFunc("/{id}/resolve", rest.RegisterInputHandler[ResolveInputRestModel](l)(db)(si)("resolve_pending_change", handleResolvePendingChange)).Methods(http.MethodPost)
			r.HandleFunc("/cancel", rest.RegisterInputHandler[CancelInputRestModel](l)(db)(si)("cancel_pending_change_for_character", handleCancelPendingChangeForCharacter)).Methods(http.MethodPost)

			// This route deliberately sits OUTSIDE the /pending-changes
			// subrouter's prefix — it is the synchronous availability check
			// (design §3.5) atlas-channel calls before it ever POSTs a
			// WORLD_TRANSFER pending-change request, so it is not itself a
			// pending-changes resource.
			router.HandleFunc("/characters/{characterId}/transfer-eligibility", registerGet("get_transfer_eligibility", handleGetTransferEligibility)).Methods(http.MethodGet)

			// The destination-free counterpart (design's OQ-7 split,
			// docs/tasks/task-227-cash-name-change-world-transfer/bug-world-transfer-eligibility-reasons.md):
			// atlas-channel's CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE handler is
			// asked before a destination world is chosen, so it cannot supply
			// destinationWorldId and cannot use the route above.
			router.HandleFunc("/characters/{characterId}/transfer-eligibility-independent", registerGet("get_transfer_eligibility_independent", handleGetTransferEligibilityIndependent)).Methods(http.MethodGet)
		}
	}
}

func handleGetPendingChanges(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ms, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetByCharacterId(characterId)
			if err != nil {
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := model.SliceMap(Transform)(model.FixedProvider(ms))()()
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

// handleGetTransferEligibility runs the full 11-gate table with no side
// effect and reports the result, so atlas-channel can render the client's
// WORLD_TRANSFER availability response without ever creating a pending
// change (design §3.5).
func handleGetTransferEligibility(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			destStr := r.URL.Query().Get("destinationWorldId")
			destVal, err := strconv.ParseUint(destStr, 10, 8)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid or missing destinationWorldId")
				return
			}
			destinationWorldId := world.Id(destVal)

			eligible, reason, err := NewProcessor(d.Logger(), d.Context(), d.DB()).
				CheckTransferEligibility(characterId, destinationWorldId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Checking transfer eligibility for character [%d] to world [%d].", characterId, destinationWorldId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res := EligibilityRestModel{
				Id:       strconv.FormatUint(uint64(characterId), 10),
				Eligible: eligible,
				Reason:   reason,
			}
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[EligibilityRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}

// handleGetTransferEligibilityIndependent runs ONLY the destination-independent
// half of the gate table (is_gm, banned, is_guild_master, in_family,
// trade_open, merchant_open, mts_listings_open), with no side effect, so
// atlas-channel's CHECK-time handler — which is asked before a destination
// world is chosen — can report a precise rejection reason instead of
// answering ALLOWED unconditionally (design's OQ-7 split).
func handleGetTransferEligibilityIndependent(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			eligible, reason, err := NewProcessor(d.Logger(), d.Context(), d.DB()).
				CheckTransferEligibilityIndependent(characterId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Checking destination-independent transfer eligibility for character [%d].", characterId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res := EligibilityRestModel{
				Id:       strconv.FormatUint(uint64(characterId), 10),
				Eligible: eligible,
				Reason:   reason,
			}
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[EligibilityRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}

func handleCreatePendingChange(d *rest.HandlerDependency, c *rest.HandlerContext, input CreateInputRestModel) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).
				CreateAndEmit(uuid.New(), characterId, input.Type, input.RequestedName, input.DestinationWorldId, input.AssetId)
			if err != nil {
				if status, reason, ok := statusForError(err); ok {
					d.Logger().WithError(err).Warnf("Rejected pending-change request for character [%d].", characterId)
					writeReasonError(w, status, reason)
					return
				}
				d.Logger().WithError(err).Errorf("Creating pending change for character [%d].", characterId)
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
}

// handleCancelPendingChange is the operator-facing cancel route: DELETE by
// record id, reason operator_cancelled. It now checks ownership -- the record
// loaded by {id} must belong to the path's {characterId} -- because it no
// longer has the field to itself: task-227's client-cancel addendum added a
// second, self-scoped cancel route (handleCancelPendingChangeForCharacter)
// reachable from a socket handler, and this route's id-only lookup had no
// ownership check at all before that (design §5.4 addendum). A mismatch is
// reported as 404, the same as an unknown id, so this route does not leak
// whether a given id exists under a different character.
func handleCancelPendingChange(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			id, err := uuid.Parse(mux.Vars(r)["id"])
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			p := NewProcessor(d.Logger(), d.Context(), d.DB())
			existing, err := p.GetById(id)
			if err != nil {
				if status, reason, ok := statusForError(err); ok {
					writeReasonError(w, status, reason)
					return
				}
				d.Logger().WithError(err).Errorf("Loading pending change [%s].", id)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			if existing.CharacterId() != characterId {
				writeReasonError(w, http.StatusNotFound, "")
				return
			}

			_, moved, err := p.ResolveAndEmit(id, StatusCancelled, "operator_cancelled")
			if err != nil {
				if status, reason, ok := statusForError(err); ok {
					writeReasonError(w, status, reason)
					return
				}
				d.Logger().WithError(err).Errorf("Cancelling pending change [%s].", id)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			if !moved {
				writeReasonError(w, http.StatusConflict, "")
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	})
}

// handleCancelPendingChangeForCharacter is the player-initiated cancel route
// (task-227 client-cancel addendum): POST .../pending-changes/cancel with the
// type in the body. It exists because the wire packet that drives it (the
// double-confirmed CANCELREQUESTS_* dialog chain riding the generic cash
// item-use op, see docs/tasks/task-227-cash-name-change-world-transfer/
// cancel-entry-point.md and cancel-confirm-semantics.md) carries no
// pending-change id -- reusing the id-based DELETE route would force
// atlas-channel into a racy GET-then-DELETE round trip. Ownership holds by
// construction here: the record is looked up by (characterId, type), not by
// id, so there is nothing to check.
//
// No pending record of the requested type is a normal race (the sweeper or
// an operator got there first), not an error -- it maps to 404, and the
// caller (atlas-channel) treats that as "nothing to cancel," not a failure.
func handleCancelPendingChangeForCharacter(d *rest.HandlerDependency, c *rest.HandlerContext, input CancelInputRestModel) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m, moved, err := NewProcessor(d.Logger(), d.Context(), d.DB()).CancelForCharacterAndType(characterId, input.Type)
			if err != nil {
				if status, reason, ok := statusForError(err); ok {
					writeReasonError(w, status, reason)
					return
				}
				d.Logger().WithError(err).Errorf("Cancelling pending [%s] change for character [%d].", input.Type, characterId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			if !moved {
				if m.Id() == uuid.Nil {
					writeReasonError(w, http.StatusNotFound, "")
					return
				}
				writeReasonError(w, http.StatusConflict, "")
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
}

// handleResolvePendingChange is the world-transfer saga's terminal-outcome
// callback (task-227 Task 13/14): APPLIED on saga success, REJECTED on saga
// failure. It reuses the same transition guard as the operator cancel route,
// so a redelivered resolve is idempotent (design §3.10) — the second call
// sees moved == false and emits nothing.
func handleResolvePendingChange(d *rest.HandlerDependency, _ *rest.HandlerContext, input ResolveInputRestModel) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(_ uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			id, err := uuid.Parse(mux.Vars(r)["id"])
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if input.Status != StatusApplied && input.Status != StatusRejected {
				server.WriteBadRequest(d.Logger(), w, "status must be APPLIED or REJECTED")
				return
			}

			_, moved, err := NewProcessor(d.Logger(), d.Context(), d.DB()).ResolveAndEmit(id, input.Status, input.Reason)
			if err != nil {
				if status, reason, ok := statusForError(err); ok {
					writeReasonError(w, status, reason)
					return
				}
				d.Logger().WithError(err).Errorf("Resolving pending change [%s] to [%s].", id, input.Status)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			if !moved {
				writeReasonError(w, http.StatusConflict, "")
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	})
}

// statusForError maps a typed processor rejection to the HTTP status the REST
// handlers write, along with the design §6 reason string (empty when the
// status alone is the whole story). It returns ok == false for an
// unrecognized (infrastructure) error, signaling the caller to fall back to
// server.WriteErrorResponse.
func statusForError(err error) (status int, reason string, ok bool) {
	var ie IneligibleError
	switch {
	case errors.As(err, &ie):
		return http.StatusUnprocessableEntity, ie.Reason, true
	case errors.Is(err, gorm.ErrRecordNotFound):
		return http.StatusNotFound, "", true
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound, "", true
	case errors.Is(err, ErrAlreadyPending):
		return http.StatusConflict, "already_pending", true
	case errors.Is(err, ErrNameReserved):
		return http.StatusConflict, "name_reserved", true
	case errors.Is(err, ErrAlreadyTerminal):
		return http.StatusConflict, "", true
	default:
		return 0, "", false
	}
}

// writeReasonError writes a JSON:API error object carrying the design §6
// reason string as its detail, mirroring server.WriteBadRequest's shape.
func writeReasonError(w http.ResponseWriter, status int, reason string) {
	type errorObject struct {
		Status string `json:"status"`
		Title  string `json:"title"`
		Detail string `json:"detail,omitempty"`
	}
	type errorDocument struct {
		Errors []errorObject `json:"errors"`
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorDocument{Errors: []errorObject{{
		Status: strconv.Itoa(status),
		Title:  http.StatusText(status),
		Detail: reason,
	}}})
}
