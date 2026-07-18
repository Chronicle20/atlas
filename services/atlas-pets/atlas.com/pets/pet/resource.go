package pet

import (
	"atlas-pets/rest"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			r := router.PathPrefix("/characters/{characterId}/pets").Subrouter()
			r.HandleFunc("", registerGet("get_pets_for_character", handleGetPetsForCharacter)).Methods(http.MethodGet)
			r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(db)(si)("create_for_character", handleCreate)).Methods(http.MethodPost)
			r = router.PathPrefix("/pets").Subrouter()
			r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(db)(si)("create", handleCreate)).Methods(http.MethodPost)
			r.HandleFunc("/{petId}", registerGet("get_pet", handleGetPet)).Methods(http.MethodGet)
		}
	}
}

func handleGetPet(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParsePetId(d.Logger(), func(petId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context(), d.DB())
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

func handleGetPetsForCharacter(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			p := NewProcessor(d.Logger(), d.Context(), d.DB())
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

// petLifespan is the standard pet lifespan (90 days), matching NewModelBuilder's
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

func handleCreate(d *rest.HandlerDependency, c *rest.HandlerContext, i RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := NewProcessor(d.Logger(), d.Context(), d.DB())
		i.Name = createPetName(i.Name)
		i.Level = createPetLevel(i.Level)
		i.Expiration = createPetExpiration(i.Expiration, time.Now())
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
