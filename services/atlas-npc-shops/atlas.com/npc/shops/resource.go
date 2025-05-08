package shops

import (
	"atlas-npc/commodities"
	"atlas-npc/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/npcs/{npcId}/shop").Subrouter()
			r.HandleFunc("", rest.RegisterHandler(l)(db)(si)("get_shop", handleGetShop)).Methods(http.MethodGet)
			r.HandleFunc("/characters", rest.RegisterHandler(l)(db)(si)("get_shop_characters", handleGetShopCharacters)).Methods(http.MethodGet)
			r.HandleFunc("/commodities", rest.RegisterInputHandler[commodities.RestModel](l)(db)(si)("add_commodity", handleAddCommodity)).Methods(http.MethodPost)
			r.HandleFunc("/commodities/{commodityId}", rest.RegisterInputHandler[commodities.RestModel](l)(db)(si)("update_commodity", handleUpdateCommodity)).Methods(http.MethodPut)
			r.HandleFunc("/commodities/{commodityId}", rest.RegisterHandler(l)(db)(si)("remove_commodity", handleRemoveCommodity)).Methods(http.MethodDelete)
		}
	}
}

func handleGetShop(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseNpcId(d.Logger(), func(npcId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context(), d.DB())
			res, err := model.Map(Transform)(p.ByNpcIdProvider(npcId))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}

func handleAddCommodity(d *rest.HandlerDependency, c *rest.HandlerContext, i commodities.RestModel) http.HandlerFunc {
	return rest.ParseNpcId(d.Logger(), func(npcId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context(), d.DB())
			// Default values for new fields
			discountRate := i.DiscountRate
			tokenItemId := i.TokenItemId
			tokenPrice := i.TokenPrice
			period := i.Period
			levelLimited := i.LevelLimit
			commodity, err := p.AddCommodity(npcId, i.TemplateId, i.MesoPrice, discountRate, tokenItemId, tokenPrice, period, levelLimited)
			if err != nil {
				d.Logger().WithError(err).Errorf("Adding commodity.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := commodities.Transform(commodity)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated)
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[commodities.RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}

func handleUpdateCommodity(d *rest.HandlerDependency, c *rest.HandlerContext, i commodities.RestModel) http.HandlerFunc {
	return rest.ParseNpcId(d.Logger(), func(npcId uint32) http.HandlerFunc {
		return rest.ParseCommodityId(d.Logger(), func(commodityId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				p := NewProcessor(d.Logger(), d.Context(), d.DB())
				// Default values for new fields
				discountRate := i.DiscountRate
				tokenItemId := i.TokenItemId
				tokenPrice := i.TokenPrice
				period := i.Period
				levelLimited := i.LevelLimit
				commodity, err := p.UpdateCommodity(commodityId, i.TemplateId, i.MesoPrice, discountRate, tokenItemId, tokenPrice, period, levelLimited)
				if err != nil {
					d.Logger().WithError(err).Errorf("Updating commodity.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				res, err := commodities.Transform(commodity)
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				w.WriteHeader(http.StatusOK)
				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[commodities.RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	})
}

func handleRemoveCommodity(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseNpcId(d.Logger(), func(npcId uint32) http.HandlerFunc {
		return rest.ParseCommodityId(d.Logger(), func(commodityId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				p := NewProcessor(d.Logger(), d.Context(), d.DB())
				err := p.RemoveCommodity(commodityId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Removing commodity.")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				w.WriteHeader(http.StatusNoContent)
			}
		})
	})
}

func handleGetShopCharacters(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseNpcId(d.Logger(), func(npcId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context(), d.DB())
			characterIds := p.GetCharactersInShop(npcId)

			res, err := model.SliceMap(TransformCharacterList)(model.FixedProvider(characterIds))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[[]CharacterListRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
		}
	})
}
