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

			// This route deliberately sits OUTSIDE the /pending-changes
			// subrouter's prefix — it is the synchronous availability check
			// (design §3.5) atlas-channel calls before it ever POSTs a
			// WORLD_TRANSFER pending-change request, so it is not itself a
			// pending-changes resource.
			router.HandleFunc("/characters/{characterId}/transfer-eligibility", registerGet("get_transfer_eligibility", handleGetTransferEligibility)).Methods(http.MethodGet)
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

// handleCancelPendingChange is the ONLY cancel path in the system. The game
// client has no SendCancel* of any kind on any version (design §4.2.1), so this
// route is operator-facing and MUST NOT be reachable from a socket handler. The
// cancel-unreachability test in atlas-channel asserts that machine-checkably.
func handleCancelPendingChange(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(_ uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			id, err := uuid.Parse(mux.Vars(r)["id"])
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			_, moved, err := NewProcessor(d.Logger(), d.Context(), d.DB()).ResolveAndEmit(id, StatusCancelled, "operator_cancelled")
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
