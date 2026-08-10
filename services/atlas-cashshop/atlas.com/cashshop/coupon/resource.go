package coupon

import (
	"atlas-cashshop/coupon/reward"
	"atlas-cashshop/rest"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

// InitResource registers the coupon admin CRUD surface (PRD §5).
//
// There is NO redeem route, and adding one is not an oversight to correct: a
// REST redeem would grant rewards to anyone who can reach the service. Every
// route below reads or edits coupon rows; none of them can grant anything.
//
// Every handler scopes to tenant.MustFromContext via the processor, and no
// route accepts a tenant id in a body — the tenant comes from the request
// headers that server.ParseTenant decodes.
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)
			registerInput := rest.RegisterInputHandler[RestModel](l)(si)

			r := router.PathPrefix("/coupons").Subrouter()
			r.HandleFunc("", registerGet("get_coupons", handleGetCoupons(db))).Methods(http.MethodGet)
			r.HandleFunc("", registerInput("create_coupon", handleCreateCoupon(db))).Methods(http.MethodPost)
			r.HandleFunc("/{couponId}", registerGet("get_coupon", handleGetCoupon(db))).Methods(http.MethodGet)
			r.HandleFunc("/{couponId}", registerInput("update_coupon", handleUpdateCoupon(db))).Methods(http.MethodPatch)
			r.HandleFunc("/{couponId}", registerGet("delete_coupon", handleDeleteCoupon(db))).Methods(http.MethodDelete)
		}
	}
}

// parseFilters reads the PRD §5 narrowings off the query string. An
// unparseable value is a client error rather than a silently ignored filter:
// quietly listing every coupon when the admin asked for one batch is how a
// bulk edit hits the wrong rows.
func parseFilters(r *http.Request) (Filters, error) {
	var f Filters
	q := r.URL.Query()

	if v := q.Get("filter[code]"); v != "" {
		f.Code = &v
	}
	if v := q.Get("filter[active]"); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			return Filters{}, errors.New("filter[active] must be true or false")
		}
		f.Active = &parsed
	}
	if v := q.Get("filter[batchId]"); v != "" {
		parsed, err := uuid.Parse(v)
		if err != nil {
			return Filters{}, errors.New("filter[batchId] must be a uuid")
		}
		f.BatchId = &parsed
	}
	for _, spec := range []struct {
		name string
		dst  **time.Time
	}{
		{"filter[expiresBefore]", &f.ExpiresBefore},
		{"filter[expiresAfter]", &f.ExpiresAfter},
	} {
		v := q.Get(spec.name)
		if v == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return Filters{}, errors.New(spec.name + " must be an RFC3339 timestamp")
		}
		*spec.dst = &parsed
	}
	return f, nil
}

func handleGetCoupons(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			f, err := parseFilters(r)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, err.Error())
				return
			}
			page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			paged, err := NewProcessor(d.Logger(), d.Context(), db).AllProvider(f, page)
			if err != nil {
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
			if err != nil {
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
		}
	}
}

func handleGetCoupon(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCouponId(d.Logger(), func(couponId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := NewProcessor(d.Logger(), d.Context(), db).GetById(couponId)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					rest.WriteError(d.Logger(), w, http.StatusNotFound, "no such coupon")
					return
				}
				if err != nil {
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				writeCoupon(d, c, w, r, m)
			}
		})
	}
}

// writeInputError maps a rejected admin request to its PRD §5 status.
//
//   - a malformed bundle, an implausible code, expiresAt <= startsAt, or
//     maxUses below the stored redemption count are 422: the request was
//     understood and refused on its content.
//   - a duplicate normalized code is 409. Codes are STORED normalized, so this
//     also catches a case/whitespace VARIANT of an existing code — the Builder
//     normalized it before the insert ever saw it.
func writeInputError(d *rest.HandlerDependency, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidCoupon), errors.Is(err, reward.ErrInvalid):
		rest.WriteError(d.Logger(), w, http.StatusUnprocessableEntity, err.Error())
	case IsDuplicateCode(err):
		rest.WriteError(d.Logger(), w, http.StatusConflict, "a coupon with this code already exists")
	default:
		server.WriteErrorResponse(d.Logger())(w)(err)
	}
}

func writeCoupon(d *rest.HandlerDependency, c *rest.HandlerContext, w http.ResponseWriter, r *http.Request, m Model) {
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

// handleCreateCoupon creates one coupon. A blank code means "generate one":
// the admin gets a code they never have to invent, drawn from crypto/rand.
func handleCreateCoupon(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context(), db)

			if err := p.ValidateRewards(input.Rewards); err != nil {
				writeInputError(d, w, err)
				return
			}

			// The id and the batch are server-owned: a created coupon belongs
			// to no batch, and its identity is minted by the insert.
			input.Id = uuid.Nil
			input.BatchId = nil

			if blankCode(input.Code) {
				generated, err := GenerateCode(DefaultGeneratedCodeLength)
				if err != nil {
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				input.Code = generated
			}

			m, err := Extract(input)
			if err != nil {
				writeInputError(d, w, err)
				return
			}

			created, err := p.Create(m)
			if err != nil {
				writeInputError(d, w, err)
				return
			}
			writeCoupon(d, c, w, r, created)
		}
	}
}

// handleUpdateCoupon applies the editable fields. code and batchId are not
// editable — the code is the identity a player types and the batch is the
// generation that produced the row — so the stored values are used regardless
// of what the body says.
func handleUpdateCoupon(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return rest.ParseCouponId(d.Logger(), func(couponId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				p := NewProcessor(d.Logger(), d.Context(), db)

				current, err := p.GetById(couponId)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					rest.WriteError(d.Logger(), w, http.StatusNotFound, "no such coupon")
					return
				}
				if err != nil {
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				if err = p.ValidateRewards(input.Rewards); err != nil {
					writeInputError(d, w, err)
					return
				}

				// SetRedemptionCount carries the STORED count into the
				// Builder, which is what makes "maxUses below the current
				// redemption count" a 422 instead of a silent shrink.
				m, err := NewBuilder(current.Code()).
					SetId(couponId).
					SetBatchId(current.BatchId()).
					SetDescription(input.Description).
					SetActive(input.Active).
					SetStartsAt(input.StartsAt).
					SetExpiresAt(input.ExpiresAt).
					SetMaxUses(input.MaxUses).
					SetRedemptionCount(current.RedemptionCount()).
					SetRewards(input.Rewards).
					Build()
				if err != nil {
					writeInputError(d, w, err)
					return
				}

				updated, err := p.Update(couponId, m)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					rest.WriteError(d.Logger(), w, http.StatusNotFound, "no such coupon")
					return
				}
				if err != nil {
					writeInputError(d, w, err)
					return
				}
				writeCoupon(d, c, w, r, updated)
			}
		})
	}
}

func handleDeleteCoupon(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCouponId(d.Logger(), func(couponId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, _ *http.Request) {
				err := NewProcessor(d.Logger(), d.Context(), db).Delete(couponId)
				switch {
				case err == nil:
					w.WriteHeader(http.StatusNoContent)
				case errors.Is(err, ErrHasRedemptions):
					// Deleting would orphan the audit trail of real grants.
					rest.WriteError(d.Logger(), w, http.StatusConflict, "coupon has redemptions and cannot be deleted")
				case errors.Is(err, gorm.ErrRecordNotFound):
					rest.WriteError(d.Logger(), w, http.StatusNotFound, "no such coupon")
				default:
					server.WriteErrorResponse(d.Logger())(w)(err)
				}
			}
		})
	}
}
