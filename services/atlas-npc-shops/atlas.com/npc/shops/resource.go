package shops

import (
	"atlas-npc/commodities"
	"atlas-npc/rest"
	"context"
	"errors"
	"net/http"
	"sort"

	"github.com/google/uuid"
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
			// Add endpoints to get and delete shops for a tenant
			router.HandleFunc("/shops", rest.RegisterHandler(l)(db)(si)("get_all_shops", handleGetAllShops)).Methods(http.MethodGet)
			router.HandleFunc("/shops", rest.RegisterHandler(l)(db)(si)("delete_all_shops", handleDeleteAllShops)).Methods(http.MethodDelete)

			r := router.PathPrefix("/npcs/{npcId}/shop").Subrouter()
			r.HandleFunc("", rest.RegisterHandler(l)(db)(si)("get_shop", handleGetShop)).Methods(http.MethodGet)
			r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(db)(si)("create_shop", handleCreateShop)).Methods(http.MethodPost)
			r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(db)(si)("update_shop", handleUpdateShop)).Methods(http.MethodPut)
			r.HandleFunc("/characters", rest.RegisterHandler(l)(db)(si)("get_shop_characters", handleGetShopCharacters)).Methods(http.MethodGet)

			// Commodities are now a relationship of shops
			r.HandleFunc("/relationships/commodities", rest.RegisterInputHandler[commodities.RestModel](l)(db)(si)("add_commodity", handleAddCommodity)).Methods(http.MethodPost)
			r.HandleFunc("/relationships/commodities", rest.RegisterHandler(l)(db)(si)("delete_all_commodities", handleDeleteAllCommodities)).Methods(http.MethodDelete)
			r.HandleFunc("/relationships/commodities/{commodityId}", rest.RegisterInputHandler[commodities.RestModel](l)(db)(si)("update_commodity", handleUpdateCommodity)).Methods(http.MethodPut)
			r.HandleFunc("/relationships/commodities/{commodityId}", rest.RegisterHandler(l)(db)(si)("remove_commodity", handleRemoveCommodity)).Methods(http.MethodDelete)
		}
	}
}

func handleGetShop(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseNpcId(d.Logger(), func(npcId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Get the shop model with decorators
			shopModel, err := NewProcessor(d.Logger(), d.Context(), d.DB()).ByNpcIdProvider(decoratorsFromInclude(d.Logger(), d.Context(), d.DB(), r)...)(npcId)()
			if err != nil {
				if errors.Is(err, ErrNotFound) {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				d.Logger().WithError(err).Errorf("Retrieving shop model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			// Transform the shop model to a REST model
			res, err := Transform(shopModel)
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

func decoratorsFromInclude(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, r *http.Request) []model.Decorator[Model] {
	query := r.URL.Query()
	includes := query["include"]
	for _, include := range includes {
		if include == "commodities" {
			return model.Decorators(NewProcessor(l, ctx, db).CommodityDecorator)
		}
	}
	return make([]model.Decorator[Model], 0)
}

func handleAddCommodity(d *rest.HandlerDependency, c *rest.HandlerContext, i commodities.RestModel) http.HandlerFunc {
	return rest.ParseNpcId(d.Logger(), func(npcId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context(), d.DB())
			// Default values for new fields
			discountRate := i.DiscountRate
			tokenTemplateId := i.TokenTemplateId
			tokenPrice := i.TokenPrice
			period := i.Period
			levelLimited := i.LevelLimit
			commodity, err := p.AddCommodity(npcId, i.TemplateId, i.MesoPrice, discountRate, tokenTemplateId, tokenPrice, period, levelLimited)
			if err != nil {
				d.Logger().WithError(err).Errorf("Adding commodity.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			res, err := commodities.Transform(commodity)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
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
				tokenTemplateId := i.TokenTemplateId
				tokenPrice := i.TokenPrice
				period := i.Period
				levelLimited := i.LevelLimit
				commodity, err := p.UpdateCommodity(commodityId, i.TemplateId, i.MesoPrice, discountRate, tokenTemplateId, tokenPrice, period, levelLimited)
				if err != nil {
					d.Logger().WithError(err).Errorf("Updating commodity.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				res, err := commodities.Transform(commodity)
				if err != nil {
					d.Logger().WithError(err).Errorf("Creating REST model.")
					server.WriteErrorResponse(d.Logger())(w)(err)
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

func handleRemoveCommodity(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseNpcId(d.Logger(), func(npcId uint32) http.HandlerFunc {
		return rest.ParseCommodityId(d.Logger(), func(commodityId uuid.UUID) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				p := NewProcessor(d.Logger(), d.Context(), d.DB())
				err := p.RemoveCommodity(commodityId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Removing commodity.")
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				w.WriteHeader(http.StatusNoContent)
			}
		})
	})
}

func handleDeleteAllCommodities(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseNpcId(d.Logger(), func(npcId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context(), d.DB())
			err := p.DeleteAllCommoditiesByNpcId(npcId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Deleting all commodities for NPC %d.", npcId)
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		}
	})
}

func handleDeleteAllShops(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := NewProcessor(d.Logger(), d.Context(), d.DB())
		err := p.DeleteAllShops()
		if err != nil {
			d.Logger().WithError(err).Errorf("Deleting all shops.")
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// handleGetShopCharacters is registry-backed (GetRegistry().GetCharactersInShop
// — an in-memory map of characters currently browsing a shop), not a DB
// query, so database.PagedQuery does not apply. GetCharactersInShop has no
// other internal caller besides this handler, so it is left unchanged and
// materialized in full here, stable-sorted (characterIds are already
// unique uint32s) for determinism, then paginate.Slice applied.
func handleGetShopCharacters(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseNpcId(d.Logger(), func(npcId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			page, err := paginate.ParseParams(r.URL.Query(), paginate.MaxPageSize, paginate.MaxPageSize)
			if err != nil {
				server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
				return
			}

			p := NewProcessor(d.Logger(), d.Context(), d.DB())
			characterIds := p.GetCharactersInShop(npcId)

			sorted := make([]uint32, len(characterIds))
			copy(sorted, characterIds)
			sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

			paged := paginate.Slice(sorted, page)

			res, err := model.SliceMap(TransformCharacterList)(model.FixedProvider(paged.Items))(model.ParallelMap())()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalPaginatedResponse[[]CharacterListRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res, paginate.EnvelopeFor(paged), r)
		}
	})
}

// handleGetAllShops is content full-table (no per-request WHERE filter), so
// database.PagedQuery is used (via AllShopsProvider/getAllShopsPaged). This
// is the one route in this task with a below-default cap: 50/250 per the
// task-117 brief (content full-table lists default to the smaller page
// size rather than the game-capped 250/250 used elsewhere).
func handleGetAllShops(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := paginate.ParseParams(r.URL.Query(), paginate.DefaultPageSize, paginate.MaxPageSize)
		if err != nil {
			server.WriteBadRequest(d.Logger(), w, "invalid page[number]/page[size]")
			return
		}

		p := NewProcessor(d.Logger(), d.Context(), d.DB())

		// Get one page of shops using the processor
		paged, err := p.GetAllShops(page, decoratorsFromInclude(d.Logger(), d.Context(), d.DB(), r)...)
		if err != nil {
			d.Logger().WithError(err).Errorf("Getting all shops.")
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}

		// Transform shop models to REST models
		restShops, err := model.SliceMap(Transform)(model.FixedProvider(paged.Items))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST models.")
			server.WriteErrorResponse(d.Logger())(w)(err)
			return
		}

		// Return the response
		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalPaginatedResponse[[]RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(restShops, paginate.EnvelopeFor(paged), r)
	}
}

func handleCreateShop(d *rest.HandlerDependency, c *rest.HandlerContext, i RestModel) http.HandlerFunc {
	return rest.ParseNpcId(d.Logger(), func(npcId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context(), d.DB())

			// Extract commodities from the REST model
			commodityModels := make([]commodities.Model, 0, len(i.Commodities))
			for _, cr := range i.Commodities {
				cm, err := commodities.Extract(cr)
				if err != nil {
					d.Logger().WithError(err).Errorf("Extracting commodity model.")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				commodityModels = append(commodityModels, cm)
			}

			// Create the shop
			shop, err := p.CreateShop(npcId, i.Recharger, commodityModels)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating shop.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			// Transform the shop model to a REST model
			restShop, err := Transform(shop)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			// Return the response
			w.WriteHeader(http.StatusCreated)
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(restShop)
		}
	})
}

func handleUpdateShop(d *rest.HandlerDependency, c *rest.HandlerContext, i RestModel) http.HandlerFunc {
	return rest.ParseNpcId(d.Logger(), func(npcId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := NewProcessor(d.Logger(), d.Context(), d.DB())

			// Extract commodities from the REST model
			commodityModels := make([]commodities.Model, 0, len(i.Commodities))
			for _, cr := range i.Commodities {
				cm, err := commodities.Extract(cr)
				if err != nil {
					d.Logger().WithError(err).Errorf("Extracting commodity model.")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				commodityModels = append(commodityModels, cm)
			}

			// Update the shop
			shop, err := p.UpdateShop(npcId, i.Recharger, commodityModels)
			if err != nil {
				d.Logger().WithError(err).Errorf("Updating shop.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			// Transform the shop model to a REST model
			restShop, err := Transform(shop)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				server.WriteErrorResponse(d.Logger())(w)(err)
				return
			}

			// Return the response
			w.WriteHeader(http.StatusOK)
			query := r.URL.Query()
			queryParams := jsonapi.ParseQueryFields(&query)
			server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(restShop)
		}
	})
}
