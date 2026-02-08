package reward

import (
	"atlas-gachapons/rest"
	"net/http"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)

			r := router.PathPrefix("/gachapons/{gachaponId}").Subrouter()
			r.HandleFunc("/rewards/select", registerGet("select_gachapon_reward", handleSelectReward)).Methods(http.MethodPost)
			r.HandleFunc("/prize-pool", registerGet("get_prize_pool", handleGetPrizePool)).Methods(http.MethodGet)
		}
	}
}

func handleSelectReward(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseGachaponId(d.Logger(), func(gachaponId string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			result, err := NewProcessor(d.Logger(), d.Context(), d.DB()).SelectReward(gachaponId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Selecting reward for gachapon [%s].", gachaponId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			rm, err := Transform(result)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
		}
	})
}

func handleGetPrizePool(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseGachaponId(d.Logger(), func(gachaponId string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			tier := r.URL.Query().Get("tier")

			pool, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetPrizePool(gachaponId, tier)
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving prize pool for gachapon [%s].", gachaponId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := model.SliceMap(Transform)(model.FixedProvider(pool))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}
