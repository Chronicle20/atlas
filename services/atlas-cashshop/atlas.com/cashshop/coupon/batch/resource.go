package batch

import (
	"atlas-cashshop/coupon"
	"atlas-cashshop/coupon/reward"
	"atlas-cashshop/rest"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

// InitResource registers the batch routes of PRD §5. A batch can be created
// and read; it cannot be edited or deleted, because the coupons it produced
// outlive the request that made them.
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)
			registerInput := rest.RegisterInputHandler[RestModel](l)(si)

			r := router.PathPrefix("/coupon-batches").Subrouter()
			r.HandleFunc("", registerGet("get_coupon_batches", handleGetBatches(db))).Methods(http.MethodGet)
			r.HandleFunc("", registerInput("create_coupon_batch", handleCreateBatch(db))).Methods(http.MethodPost)
			r.HandleFunc("/{batchId}", registerGet("get_coupon_batch", handleGetBatch(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetBatches(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}
			paged, err := NewProcessor(d.Logger(), d.Context(), db).AllProvider(page)
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

func handleGetBatch(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseBatchId(d.Logger(), func(batchId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				m, err := NewProcessor(d.Logger(), d.Context(), db).GetById(batchId)
				if errors.Is(err, gorm.ErrRecordNotFound) {
					rest.WriteError(d.Logger(), w, http.StatusNotFound, "no such coupon batch")
					return
				}
				if err != nil {
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				res, terr := model.Map(Transform)(model.FixedProvider(m))()
				if terr != nil {
					server.WriteErrorResponse(d.Logger())(w)(terr)
					return
				}
				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}

// handleCreateBatch generates the whole batch or none of it. The generated
// plaintext codes are returned on THIS response only — the batch row does not
// store them, and no later GET can serve them.
func handleCreateBatch(db *gorm.DB) rest.InputHandler[RestModel] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			in := ExtractInput(input)
			if in.Length == 0 {
				in.Length = coupon.DefaultGeneratedCodeLength
			}

			// The commodity check is the coupon processor's, because it is the
			// same "does this serial exist?" question a single-coupon create
			// asks, and one answer keeps the two 422s consistent.
			if err := coupon.NewProcessor(d.Logger(), d.Context(), db).ValidateRewards(in.Rewards); err != nil {
				writeBatchInputError(d, w, err)
				return
			}

			m, codes, err := NewProcessor(d.Logger(), d.Context(), db).Generate(in)
			if err != nil {
				writeBatchInputError(d, w, err)
				return
			}

			res, err := model.Map(Transform)(model.FixedProvider(m))()
			if err != nil {
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}
			res.Codes = codes

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	}
}

// writeBatchInputError maps a rejected generation request to its status. An
// over-long prefix+length combination, a count of zero, a malformed bundle and
// an unknown commodity serial are all 422 — refused on content, understood as
// a request.
func writeBatchInputError(d *rest.HandlerDependency, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidBatch), errors.Is(err, reward.ErrInvalid), errors.Is(err, coupon.ErrInvalidCoupon):
		rest.WriteError(d.Logger(), w, http.StatusUnprocessableEntity, err.Error())
	default:
		server.WriteErrorResponse(d.Logger())(w)(err)
	}
}
