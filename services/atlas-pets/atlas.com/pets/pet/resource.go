package pet

import (
	"atlas-pets/rest"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	petconst "github.com/Chronicle20/atlas/libs/atlas-constants/pet"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)
			registerInput := rest.RegisterInputHandler[RestModel](l)(si)
			r := router.PathPrefix("/characters/{characterId}/pets").Subrouter()
			r.HandleFunc("", registerGet("get_pets_for_character", handleGetPetsForCharacter(db))).Methods(http.MethodGet)
			r.HandleFunc("", registerInput("create_for_character", handleCreate(db))).Methods(http.MethodPost)
			r = router.PathPrefix("/pets").Subrouter()
			r.HandleFunc("", registerInput("create", handleCreate(db))).Methods(http.MethodPost)
			r.HandleFunc("/{petId}", registerGet("get_pet", handleGetPet(db))).Methods(http.MethodGet)
			r.HandleFunc("/{petId}", registerInput("update_pet", handleUpdate(db))).Methods(http.MethodPatch)
		}
	}
}

func handleGetPet(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParsePetId(d.Logger(), func(petId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				p := NewProcessor(d.Logger(), d.Context(), db)
				res, err := model.Map(Transform(d.Context()))(p.ByIdProvider(petId))()
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

func handleGetPetsForCharacter(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
					return
				}

				p := NewProcessor(d.Logger(), d.Context(), db)
				paged, err := p.ByOwnerIdPagedProvider(characterId, page)()
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to locate pets for character [%d].", characterId)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				res, err := model.SliceMap(Transform(d.Context()))(model.FixedProvider(paged.Items))(model.ParallelMap())()
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
}

// createPetName defaults a missing pet name. Pets granted through the generic
// inventory/award path (e.g. the GM @award item command) supply no name, but the
// model requires one ("name is required"). The player-facing cash-shop path
// resolves the WZ name from atlas-data and passes it explicitly; the generic
// award path does not, so an empty name falls back to "Pet".
func createPetName(provided string) string {
	if provided != "" {
		return provided
	}
	return "Pet"
}

// createPetLevel defaults a pet's level for creation. The generic inventory/award
// path POSTs a bare pet (level 0), which fails the model's "level must be between
// 1 and 30" check; a new pet starts at level 1 (mirroring the processor's
// new-pet defaults). A valid level (1-30) is preserved.
func createPetLevel(provided byte) byte {
	if provided < 1 || provided > 30 {
		return 1
	}
	return provided
}

// petLifespan is the standard pet lifespan (90 days), matching NewBuilder's
// default and the evolution reset.
const petLifespan = 2160 * time.Hour

// createPetExpiration defaults a pet's expiration for creation. The generic
// inventory/award path POSTs a bare pet with a zero/epoch expiration, which would
// create the pet already-expired ("dried up"). A zero expiration becomes
// now + the standard lifespan; a provided expiration is preserved.
func createPetExpiration(provided time.Time, now time.Time) time.Time {
	if provided.IsZero() {
		return now.Add(petLifespan)
	}
	return provided
}

// SlotUnspawned is the slot value for a pet that is not currently out. Slots
// 0..2 are the three active pet positions.
const SlotUnspawned = int8(-1)

// createPetSlot forces a newly created pet to start unspawned.
//
// Slot is a plain int8 on the wire, so an absent "slot" field is indistinguishable
// from a deliberate 0 — and 0 means "spawned in the first pet position". Neither
// producer sends the field (atlas-cashshop's pet RestModel has no Slot at all;
// atlas-inventory posts a bare model), so every pet was being created already
// spawned. Two purchases then both sat in slot 0, which is a state Spawn itself
// can never produce: the client shows one pet it never summoned and cannot
// dismiss, and the hunger task decays both, so slot 0's displayed fullness
// alternates between two different pets.
//
// Spawning is an explicit action (Processor.Spawn, which assigns the slot and
// shifts existing pets); creation must never confer it. This mirrors
// createPetName / createPetLevel / createPetExpiration, which exist because the
// same bare-POST path under-specifies those fields too.
func createPetSlot() int8 {
	return SlotUnspawned
}

func handleCreate(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, i RestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context(), db)
			i.Name = createPetName(i.Name)
			i.Level = createPetLevel(i.Level)
			i.Expiration = createPetExpiration(i.Expiration, time.Now())
			i.Slot = createPetSlot()
			ip, err := model.Map(Extract)(model.FixedProvider(i))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to create model from input.")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			pm, err := p.CreateAndEmit(ip)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to create model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			res, err := model.Map(Transform(d.Context()))(model.FixedProvider(pm))()
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

// handleUpdate is the operator surface for correcting a pet name without a
// direct DB write. The gameplay rename path is the RENAME Kafka command driven
// by the pet_name_tag_use saga -- atlas-channel never calls this endpoint
// (PRD §5.1). `name` is the only writable attribute; every other field on the
// inbound RestModel is ignored.
func handleUpdate(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, i RestModel) http.HandlerFunc {
		return rest.ParsePetId(d.Logger(), func(petId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				name := petconst.NormalizeName(i.Name)
				if err := petconst.ValidateName(name); err != nil {
					d.Logger().WithError(err).Warnf("Rejecting PATCH of pet [%d]: invalid name [%s].", petId, i.Name)
					server.WriteBadRequest(d.Logger(), w, err.Error())
					return
				}

				p := NewProcessor(d.Logger(), d.Context(), db)
				existing, err := p.GetById(petId)
				if err != nil {
					d.Logger().WithError(err).Warnf("Unable to locate pet [%d].", petId)
					if errors.Is(err, gorm.ErrRecordNotFound) {
						writeNotFound(d.Logger(), w, "pet not found")
						return
					}
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				// The owner is taken from the stored row, never from the request:
				// the processor's ownership check would otherwise be trivially
				// satisfiable by a caller supplying whatever ownerId it liked. This
				// makes the check near-tautological through this handler -- it is
				// reachable only if the pet's owner changes between this read and
				// the processor's re-read inside its own transaction.
				if err = p.RenameAndEmit(uuid.New(), petId, existing.OwnerId(), name); err != nil {
					d.Logger().WithError(err).Warnf("Unable to rename pet [%d].", petId)
					if errors.Is(err, ErrNotOwner) {
						writeForbidden(d.Logger(), w, "pet is not owned by character")
						return
					}
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				updated, err := p.GetById(petId)
				if err != nil {
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				res, err := model.Map(Transform(d.Context()))(model.FixedProvider(updated))()
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

// operatorErrorObject / operatorErrorBody mirror atlas-rest/server's
// unexported JSON:API error shape ({"errors":[{"status","title","detail"}]}).
// handleUpdate is the operator PATCH surface this branch introduced; it needs
// 404 (pet not found) and 403 (rename rejected for non-ownership) status
// codes that server.WriteErrorResponse does not produce -- that helper only
// distinguishes transient-503 from 500, and installing a not-found classifier
// there would change error mapping for every service in the repo. Kept local
// to this handler rather than touching libs/atlas-rest or any other handler
// in this file (see docs/tasks/task-224-pet-name-tag/audit-backend-guidelines.md).
type operatorErrorObject struct {
	Status string `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

type operatorErrorBody struct {
	Errors []operatorErrorObject `json:"errors"`
}

func writeOperatorError(l logrus.FieldLogger, w http.ResponseWriter, status int, title string, detail string) {
	body, err := json.Marshal(operatorErrorBody{Errors: []operatorErrorObject{{
		Status: strconv.Itoa(status),
		Title:  title,
		Detail: detail,
	}}})
	if err != nil {
		l.WithError(err).Errorf("Unable to marshal error response.")
		body = []byte(`{"errors":[{"status":"` + strconv.Itoa(status) + `","title":"` + title + `"}]}`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		l.WithError(err).Errorf("Unable to write error response.")
	}
}

// writeNotFound writes a JSON:API error object with HTTP 404.
func writeNotFound(l logrus.FieldLogger, w http.ResponseWriter, detail string) {
	writeOperatorError(l, w, http.StatusNotFound, "Not Found", detail)
}

// writeForbidden writes a JSON:API error object with HTTP 403.
func writeForbidden(l logrus.FieldLogger, w http.ResponseWriter, detail string) {
	writeOperatorError(l, w, http.StatusForbidden, "Forbidden", detail)
}
