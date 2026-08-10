package redemption

import (
	"atlas-cashshop/rest"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

// InitResource registers the two READ-ONLY audit routes of PRD §5. Both are
// registered on the top-level router rather than under a shared subrouter, so
// package coupon's /coupons subrouter and this package's nested path stay
// independent of registration order.
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)
			router.HandleFunc("/coupons/{couponId}/redemptions", registerGet("get_coupon_redemptions", handleGetByCoupon(db))).Methods(http.MethodGet)
			router.HandleFunc("/coupon-redemptions", registerGet("get_redemptions", handleGetByAccount(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetByCoupon(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCouponId(d.Logger(), func(couponId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
					return
				}
				paged, err := NewProcessor(d.Logger(), d.Context(), db).ByCouponId(couponId, page)
				if err != nil {
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				writePage(d, c, w, r, paged)
			}
		})
	}
}

// handleGetByAccount REQUIRES filter[accountId]. There is deliberately no
// unfiltered arm: a bare listing of every redemption in the tenant is an
// audit dump nothing in PRD §5 asks for, and defaulting to it when the filter
// is missing would hand one out on a typo.
func handleGetByAccount(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			raw := r.URL.Query().Get("filter[accountId]")
			if raw == "" {
				server.WriteBadRequest(d.Logger(), w, "filter[accountId] is required")
				return
			}
			parsed, err := strconv.ParseUint(raw, 10, 32)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "filter[accountId] must be a positive integer")
				return
			}

			page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			paged, err := NewProcessor(d.Logger(), d.Context(), db).ByAccountId(uint32(parsed), page)
			if err != nil {
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			writePage(d, c, w, r, paged)
		}
	}
}

func writePage(d *rest.HandlerDependency, c *rest.HandlerContext, w http.ResponseWriter, r *http.Request, paged model.Paged[Model]) {
	res, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
	if err != nil {
		server.WriteErrorResponse(d.Logger())(w)(err)
		return
	}
	query := r.URL.Query()
	queryParams := jsonapi.ParseQueryFields(&query)
	server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
}
