package equipable

import (
	"atlas-equipables/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

func InitResource(si jsonapi.ServerInformation, db *gorm.DB) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(db)(si)
		registerInput := rest.RegisterInputHandler[RestModel](l)(db)(si)
		registerDelete := rest.RegisterHandler(l)(db)(si)

		r := router.PathPrefix("/equipables").Subrouter()
		r.HandleFunc("", registerInput("create_random_equipment", handleCreateRandomEquipment)).Queries("random", "{random}").Methods(http.MethodPost)
		r.HandleFunc("", registerInput("create_equipment", handleCreateEquipment)).Methods(http.MethodPost)
		r.HandleFunc("/{equipmentId}", registerGet("get_equipment", handleGetEquipment)).Methods(http.MethodGet)
		r.HandleFunc("/{equipmentId}", registerInput("update_equipment", handleUpdateEquipment)).Methods(http.MethodPatch)
		r.HandleFunc("/{equipmentId}", registerDelete("delete_equipment", handleDeleteEquipment)).Methods(http.MethodDelete)
	}
}

func handleCreateRandomEquipment(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		e, err := NewProcessor(d.Logger(), d.Context(), d.DB()).CreateRandomAndEmit(input.ItemId)
		if err != nil {
			d.Logger().WithError(err).Errorf("Cannot create equipable.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		rm, err := model.Map(Transform)(model.FixedProvider(e))()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
	}
}

func handleCreateEquipment(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		i, err := model.Map(Extract)(model.FixedProvider(input))()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		e, err := NewProcessor(d.Logger(), d.Context(), d.DB()).CreateAndEmit(i)
		if err != nil {
			d.Logger().WithError(err).Errorf("Cannot create equipable.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		rm, err := model.Map(Transform)(model.FixedProvider(e))()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
	}
}

func handleGetEquipment(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseEquipmentId(d.Logger(), func(equipmentId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			e, err := NewProcessor(d.Logger(), d.Context(), d.DB()).GetById(equipmentId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to retrieve equipable %d.", equipmentId)
				w.WriteHeader(http.StatusNotFound)
				return
			}
			rm, err := model.Map(Transform)(model.FixedProvider(e))()
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

func handleUpdateEquipment(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		i, err := model.Map(Extract)(model.FixedProvider(input))()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		e, err := NewProcessor(d.Logger(), d.Context(), d.DB()).UpdateAndEmit(i)
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to update equipable %d.", i.Id())
			w.WriteHeader(http.StatusNotFound)
			return
		}
		rm, err := model.Map(Transform)(model.FixedProvider(e))()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		query := r.URL.Query()
		queryParams := jsonapi.ParseQueryFields(&query)
		server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(rm)
	}
}

func handleDeleteEquipment(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseEquipmentId(d.Logger(), func(equipmentId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := NewProcessor(d.Logger(), d.Context(), d.DB()).DeleteByIdAndEmit(equipmentId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to delete equipable %d.", equipmentId)
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	})
}
