package equipment

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

const (
	createRandomEquipment = "create_random_equipment"
	createEquipment       = "create_equipment"
	getEquipment          = "get_equipment"
	deleteEquipment       = "delete_equipment"
)

func InitResource(si jsonapi.ServerInformation, db *gorm.DB) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(db)(si)
		registerCreate := rest.RegisterInputHandler[RestModel](l)(db)(si)
		registerDelete := rest.RegisterHandler(l)(db)(si)

		r := router.PathPrefix("/equipables").Subrouter()
		r.HandleFunc("", registerCreate(createRandomEquipment, handleCreateRandomEquipment)).Queries("random", "{random}").Methods(http.MethodPost)
		r.HandleFunc("", registerCreate(createEquipment, handleCreateEquipment)).Methods(http.MethodPost)
		r.HandleFunc("/{equipmentId}", registerGet(getEquipment, handleGetEquipment)).Methods(http.MethodGet)
		r.HandleFunc("/{equipmentId}", registerDelete(deleteEquipment, handleDeleteEquipment)).Methods(http.MethodDelete)
	}
}

func handleDeleteEquipment(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseEquipmentId(d.Logger(), func(equipmentId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := DeleteById(d.Logger())(d.DB())(d.Context())(equipmentId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to delete equipment %d.", equipmentId)
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	})
}

func handleCreateRandomEquipment(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		e, err := CreateRandom(d.Logger())(d.DB())(d.Context())(input.ItemId)
		if err != nil {
			d.Logger().WithError(err).Errorf("Cannot create equipment.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.Map(Transform)(model.FixedProvider(e))()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
	}
}

func handleCreateEquipment(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		e, err := Create(d.Logger())(d.DB())(d.Context())(input.ItemId, input.Strength, input.Dexterity, input.Intelligence, input.Luck,
			input.HP, input.MP, input.WeaponAttack, input.MagicAttack, input.WeaponDefense, input.MagicDefense, input.Accuracy,
			input.Avoidability, input.Hands, input.Speed, input.Jump, input.Slots)
		if err != nil {
			d.Logger().WithError(err).Errorf("Cannot create equipment.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.Map(Transform)(model.FixedProvider(e))()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
	}
}

func handleGetEquipment(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseEquipmentId(d.Logger(), func(equipmentId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			e, err := GetById(d.Logger())(d.DB())(d.Context())(equipmentId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to retrieve equipment %d.", equipmentId)
				w.WriteHeader(http.StatusNotFound)
				return
			}
			res, err := model.Map(Transform)(model.FixedProvider(e))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}
