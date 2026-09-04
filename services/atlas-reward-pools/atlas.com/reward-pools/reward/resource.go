package reward

import (
	"atlas-reward-pools/rest"
	"errors"
	"net/http"
	"sort"

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
			registerGet := rest.RegisterHandler(l)(si)

			r := router.PathPrefix("/gachapons/{gachaponId}").Subrouter()
			r.HandleFunc("/rewards/select", registerGet("select_gachapon_reward", handleSelectReward(db))).Methods(http.MethodPost)
			r.HandleFunc("/prize-pool", registerGet("get_prize_pool", handleGetPrizePool(db))).Methods(http.MethodGet)
		}
	}
}

func handleSelectReward(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseGachaponId(d.Logger(), func(gachaponId string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				result, err := NewProcessor(d.Logger(), d.Context(), db).SelectReward(gachaponId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Selecting reward for gachapon [%s].", gachaponId)
					if errors.Is(err, ErrEmptyPool) {
						// The pool exists but has nothing to give. 409, not 500:
						// the caller (atlas-cashshop) distinguishes this from a
						// missing pool to log POOL_EMPTY vs POOL_MISSING.
						w.WriteHeader(http.StatusConflict)
						return
					}
					if errors.Is(err, gorm.ErrRecordNotFound) {
						// No pool row matches gachaponId at all. 404, not 500:
						// distinct from ErrEmptyPool above so the caller can log
						// POOL_MISSING vs POOL_EMPTY.
						w.WriteHeader(http.StatusNotFound)
						return
					}
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				rm, err := Transform(result)
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
			}
		})
	}
}

// handleGetPrizePool materializes the full merged prize pool (machine items
// + global items for the requested tier(s)) and paginates it in-memory: the
// pool is a computed merge across two tables, not a single Where-filtered
// query, so it cannot be pushed down to database.PagedQuery. The merged
// result has no natural single-column ordering, so it is stable-sorted by
// (tier, itemId, gachaponId) before slicing to make paging deterministic.
func handleGetPrizePool(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseGachaponId(d.Logger(), func(gachaponId string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				tier := r.URL.Query().Get("tier")

				page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
					return
				}

				pool, err := NewProcessor(d.Logger(), d.Context(), db).GetPrizePool(gachaponId, tier)
				if err != nil {
					d.Logger().WithError(err).Errorf("Retrieving prize pool for gachapon [%s].", gachaponId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				res, err := model.SliceMap(Transform)(model.FixedProvider(pool))(model.ParallelMap())()
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				sort.SliceStable(res, func(i, j int) bool {
					if res[i].Tier != res[j].Tier {
						return res[i].Tier < res[j].Tier
					}
					if res[i].ItemId != res[j].ItemId {
						return res[i].ItemId < res[j].ItemId
					}
					return res[i].GachaponId < res[j].GachaponId
				})

				paged := paginate.Slice(res, page)
				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
			}
		})
	}
}
