package item

import (
	"atlas-gachapons/rest"
	"net/http"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			registerInput := rest.RegisterInputHandler[RestModel](l)(db)(si)

			r := router.PathPrefix("/gachapons/{gachaponId}/items").Subrouter()
			r.HandleFunc("", registerGet("get_gachapon_items", handleGetItems)).Methods(http.MethodGet)
			r.HandleFunc("", registerInput("create_gachapon_item", handleCreateItem)).Methods(http.MethodPost)
			r.HandleFunc("/{itemId}", registerGet("delete_gachapon_item", handleDeleteItem)).Methods(http.MethodDelete)
		}
	}
}

func handleGetItems(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseGachaponId(d.Logger(), func(gachaponId string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			tier := r.URL.Query().Get("tier")

			var ms []Model
			var err error
			p := NewProcessor(d.Logger(), d.Context(), d.DB())
			if tier != "" {
				ms, err = p.GetByGachaponIdAndTier(gachaponId, tier)()
			} else {
				ms, err = p.GetByGachaponId(gachaponId)()
			}
			if err != nil {
				d.Logger().WithError(err).Errorf("Retrieving items for gachapon [%s].", gachaponId)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			res, err := model.SliceMap(Transform)(model.FixedProvider(ms))(model.ParallelMap())()
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

func handleCreateItem(d *rest.HandlerDependency, c *rest.HandlerContext, rm RestModel) http.HandlerFunc {
	return rest.ParseGachaponId(d.Logger(), func(gachaponId string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			t := tenant.MustFromContext(d.Context())
			m, err := NewBuilder(t.Id(), 0).
				SetGachaponId(gachaponId).
				SetItemId(rm.ItemId).
				SetQuantity(rm.Quantity).
				SetTier(rm.Tier).
				Build()
			if err != nil {
				d.Logger().WithError(err).Errorf("Building item model.")
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			err = NewProcessor(d.Logger(), d.Context(), d.DB()).Create(m)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating gachapon item.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated)
		}
	})
}

func handleDeleteItem(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseGachaponId(d.Logger(), func(_ string) http.HandlerFunc {
		return rest.ParseItemId(d.Logger(), func(itemId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := NewProcessor(d.Logger(), d.Context(), d.DB()).Delete(itemId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Deleting gachapon item [%d].", itemId)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}
		})
	})
}
