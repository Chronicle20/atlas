package ring

import (
	"atlas-cashshop/rest"
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	restserver "github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// byCharacterIdPagedProvider pages every ring half a character holds, for
// GET /rings?filter[characterId]=. Defined locally rather than in
// provider.go: this file is additive REST surface over the already-shipped
// CreatePair/GetByCharacterId/GetById (task-240 task 18), and stays out of
// that file while its own review is in flight.
func byCharacterIdPagedProvider(t tenant.Model, characterId uint32, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](
			db.Where("tenant_id = ? AND character_id = ?", t.Id(), characterId).Order("created_at, id"), page)
	}
}

// InitResource registers the two READ-ONLY routes PRD §5.4 leaves open:
// GET /rings (filtered by character) and GET /rings/{ringId}. This closes
// that open question in favor of atlas-cashshop, which already owns the
// write path via cashshop.PurchaseRingAndEmit.
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) restserver.RouteInitializer {
	return func(db *gorm.DB) restserver.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)
			router.HandleFunc("/rings", registerGet("get_rings", handleGetRings(db))).Methods(http.MethodGet)
			router.HandleFunc("/rings/{ringId}", registerGet("get_ring", handleGetRing(db))).Methods(http.MethodGet)
		}
	}
}

// handleGetRings REQUIRES filter[characterId]. There is deliberately no
// unfiltered arm: a bare listing of every ring pair half in the tenant is an
// audit dump nothing in PRD §5.4 asks for, mirroring
// coupon/redemption/resource.go's handleGetByAccount.
func handleGetRings(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			raw := r.URL.Query().Get("filter[characterId]")
			if raw == "" {
				restserver.WriteBadRequest(d.Logger(), w, "filter[characterId] is required")
				return
			}
			parsed, err := strconv.ParseUint(raw, 10, 32)
			if err != nil {
				restserver.WriteBadRequest(d.Logger(), w, "filter[characterId] must be a positive integer")
				return
			}

			page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				restserver.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			t := tenant.MustFromContext(d.Context())
			ep := byCharacterIdPagedProvider(t, uint32(parsed), page)(db.WithContext(d.Context()))
			paged, err := model.MapPaged(Make)(ep)(model.ParallelMap())()
			if err != nil {
				restserver.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				restserver.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			restserver.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
		}
	}
}

func handleGetRing(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return restserver.ParseUUIDId(d.Logger(), "ringId", func(ringId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				t := tenant.MustFromContext(d.Context())
				m, err := GetById(db.WithContext(d.Context()), t.Id(), ringId)
				// A ring belonging to another tenant resolves to
				// gorm.ErrRecordNotFound (GetById scopes on tenant_id), the
				// same as a genuinely unknown id -- mirrors
				// coupon/resource.go's handleGetCoupon, whose 404 mapping
				// this route did not have (task 24a item 2's HTTP-level
				// cross-tenant isolation test caught the gap: a 500 leaked
				// server-error noise for what is an ordinary 404).
				if errors.Is(err, gorm.ErrRecordNotFound) {
					rest.WriteError(d.Logger(), w, http.StatusNotFound, "no such ring")
					return
				}
				if err != nil {
					restserver.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				res, err := model.Map(Transform)(model.FixedProvider(m))()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					restserver.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				restserver.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}
