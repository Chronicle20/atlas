package parcel

import (
	"atlas-parcel/rest"
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

// InitResource registers the parcel custody REST surface — the read API
// reached AFTER a send request has already been validated and accepted
// (Create/Receive/Discard are driven by the Kafka command consumer, task-15,
// not by this REST surface):
//
//   - GET /parcels?filter[recipientId]=&filter[worldId]=&filter[status]= — a
//     recipient's mailbox in a world
//   - GET /parcels?filter[senderId]=&filter[status]=                    — a
//     sender's still-in-flight outbound parcels
//   - GET /parcels/{parcelId}                                            — a
//     single parcel by id
//   - GET /characters/{characterId}/parcel-status                       — a
//     narrow "does this character have a pending parcel" lookup (task-26's
//     world-transfer gate 12), one round trip instead of a full mailbox fetch
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)

			pr := router.PathPrefix("/parcels").Subrouter()
			pr.HandleFunc("", registerGet("get_parcels", handleGetParcels)).Methods(http.MethodGet)
			pr.HandleFunc("/{parcelId}", registerGet("get_parcel", handleGetParcel)).Methods(http.MethodGet)

			cr := router.PathPrefix("/characters/{characterId}").Subrouter()
			cr.HandleFunc("/parcel-status", registerGet("get_character_parcel_status", handleGetParcelStatus)).Methods(http.MethodGet)
		}
	}
}

// handleGetParcels lists either a recipient's mailbox or a sender's
// still-in-flight outbound parcels, depending on which filter is supplied.
// filter[recipientId] and filter[senderId] are mutually exclusive entry
// points into the two Processor reads that exist (GetForRecipient,
// GetPendingForSender) — both of which only ever surface StatusPending
// parcels, so filter[status], when supplied, must be "pending"; anything
// else is a clean 400, never a disconnect. filter[worldId] is REQUIRED with
// filter[recipientId] — world 0 is an ordinary real world, not a sentinel,
// so a missing filter[worldId] must never silently default to it (a tenant
// has many worlds; this is the third instance of this exact mis-scoping
// risk in the plan — task-2's provider WHERE clause and task-3's
// HasInFlight both had the same shape of finding).
func handleGetParcels(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		if v := q.Get("filter[status]"); v != "" && v != StatusPending {
			server.WriteBadRequest(d.Logger(), w, "filter[status] must be \"pending\"")
			return
		}

		p := NewProcessor(d.Logger(), d.Context(), d.DB())

		switch {
		case q.Get("filter[recipientId]") != "":
			recipientId, err := strconv.ParseUint(q.Get("filter[recipientId]"), 10, 32)
			if err != nil {
				d.Logger().WithError(err).Warnf("Unable to parse filter[recipientId].")
				server.WriteBadRequest(d.Logger(), w, "filter[recipientId] must be a uint32")
				return
			}

			v := q.Get("filter[worldId]")
			if v == "" {
				d.Logger().Warnf("Parcel list request for recipient [%d] omitted filter[worldId].", recipientId)
				server.WriteBadRequest(d.Logger(), w, "filter[worldId] is required")
				return
			}
			parsed, werr := strconv.ParseUint(v, 10, 8)
			if werr != nil {
				d.Logger().WithError(werr).Warnf("Unable to parse filter[worldId].")
				server.WriteBadRequest(d.Logger(), w, "filter[worldId] must be a byte")
				return
			}
			worldId := world.Id(byte(parsed))

			ms, err := p.GetForRecipient(uint32(recipientId), worldId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving parcels for recipient [%d].", recipientId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			writeParcels(d, c, w, r, ms)
			return

		case q.Get("filter[senderId]") != "":
			senderId, err := strconv.ParseUint(q.Get("filter[senderId]"), 10, 32)
			if err != nil {
				d.Logger().WithError(err).Warnf("Unable to parse filter[senderId].")
				server.WriteBadRequest(d.Logger(), w, "filter[senderId] must be a uint32")
				return
			}

			ms, err := p.GetPendingForSender(uint32(senderId))
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving parcels for sender [%d].", senderId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			writeParcels(d, c, w, r, ms)
			return

		default:
			d.Logger().Warnf("Parcel list request supplied neither filter[recipientId] nor filter[senderId].")
			server.WriteBadRequest(d.Logger(), w, "one of filter[recipientId] or filter[senderId] is required")
			return
		}
	}
}

func writeParcels(d *rest.HandlerDependency, c *rest.HandlerContext, w http.ResponseWriter, r *http.Request, ms []Model) {
	res, err := model.SliceMap(Transform)(model.FixedProvider(ms))(model.ParallelMap())()
	if err != nil {
		d.Logger().WithError(err).Errorf("Creating REST model.")
		server.WriteErrorResponse(d.Logger())(w)(err)
		return
	}

	query := r.URL.Query()
	queryParams := jsonapi.ParseQueryFields(&query)
	server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
}

// handleGetParcel retrieves a single parcel by id. A malformed uuid or an id
// with no row is rejected cleanly (400 / 404) — never a disconnect.
func handleGetParcel(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseParcelId(d.Logger(), func(parcelIdStr string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			parcelId, err := uuid.Parse(parcelIdStr)
			if err != nil {
				d.Logger().WithError(err).Warnf("Unable to parse parcelId [%s].", parcelIdStr)
				server.WriteBadRequest(d.Logger(), w, "parcelId must be a uuid")
				return
			}

			m, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetById(parcelId)
			if errors.Is(err, ErrNotFound) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving parcel [%s].", parcelId)
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

// handleGetParcelStatus answers "does this character have a pending
// parcel" — the narrow round trip task-26's world-transfer gate 12 calls,
// rather than a full mailbox fetch.
func handleGetParcelStatus(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			inFlight, err := NewProcessor(d.Logger(), d.Context(), d.DB()).HasInFlight(characterId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving parcel status for character [%d].", characterId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := transformParcelStatus(characterId, inFlight)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[parcelStatusRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}
