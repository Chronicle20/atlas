package equipslot

import (
	"atlas-character/rest"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

// InitResource wires the read/write surface atlas-cashshop and atlas-channel
// need for a character's equip-slot extensions: GET the active extensions
// (R3) and POST to extend one (task-240 task 23, R2 -- the write side this
// package's original InitResource doc comment deferred).
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			r := router.PathPrefix("/characters/{characterId}/equip-slot-extensions").Subrouter()
			r.HandleFunc("", registerGet("get_equip_slot_extensions", handleGetEquipSlotExtensions)).Methods(http.MethodGet)
			r.HandleFunc("", rest.RegisterInputHandler[ExtendInputRestModel](l)(db)(si)("extend_equip_slot", handleExtendEquipSlot)).Methods(http.MethodPost)
		}
	}
}

func handleGetEquipSlotExtensions(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ms, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetActive(characterId)
			if err != nil {
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := TransformSlice(ms)
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

// handleExtendEquipSlot is the write side task 22's InitResource doc comment
// deferred (task-240 task 23, R2). SlotIndex on the input is the caller's
// already-resolved Atlas canonical position (R1) -- this handler persists it
// as given, it does not resolve or invent it.
func handleExtendEquipSlot(d *rest.HandlerDependency, c *rest.HandlerContext, input ExtendInputRestModel) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			period := time.Duration(input.Days) * 24 * time.Hour
			p := NewProcessor(d.Logger(), d.Context(), d.DB())
			if _, err := p.Extend(characterId, input.SlotIndex, period, input.TransactionId); err != nil {
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			// Re-read the row Extend just wrote so the response carries its
			// real row id -- Extend returns only the resulting expiry
			// (processor.go), not the Entity/Model it upserted.
			ms, err := p.GetActive(characterId)
			if err != nil {
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			var m Model
			for _, cand := range ms {
				if cand.SlotIndex() == input.SlotIndex {
					m = cand
					break
				}
			}

			res, err := Transform(m)
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
