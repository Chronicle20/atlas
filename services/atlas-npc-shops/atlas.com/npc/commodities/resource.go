package commodities

import (
	"atlas-npc/rest"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/commodities/items").Subrouter()
			r.HandleFunc("/{itemId}", rest.RegisterHandler(l)(db)(si)("get_commodities_by_item", handleGetCommoditiesByItem)).Methods(http.MethodGet)
		}
	}
}

func handleGetCommoditiesByItem(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		itemIdRaw := vars["itemId"]
		itemId, err := strconv.ParseUint(itemIdRaw, 10, 32)
		if err != nil {
			d.Logger().WithError(err).Errorf("Error parsing itemId as uint32")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var entities []Entity
		if err := d.DB().WithContext(d.Context()).
			Where("template_id = ?", uint32(itemId)).
			Find(&entities).Error; err != nil {
			d.Logger().WithError(err).Errorf("Unable to retrieve commodities for itemId=%d.", itemId)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res := make([]CommodityByItemRestModel, 0, len(entities))
		for _, e := range entities {
			res = append(res, CommodityByItemRestModel{
				Id:              e.Id,
				NpcId:           e.NpcId,
				TemplateId:      e.TemplateId,
				MesoPrice:       e.MesoPrice,
				DiscountRate:    e.DiscountRate,
				TokenTemplateId: e.TokenTemplateId,
				TokenPrice:      e.TokenPrice,
				Period:          e.Period,
				LevelLimit:      e.LevelLimit,
			})
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[[]CommodityByItemRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
	}
}
