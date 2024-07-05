package statistics

import (
	"atlas-data/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"net/http"
)

const (
	getEquipmentStatistics = "get_equipment_statistics"
)

func InitResource(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(router *mux.Router, l logrus.FieldLogger) {
		registerGet := rest.RegisterHandler(l)(si)

		r := router.PathPrefix("/equipment").Subrouter()
		r.HandleFunc("/{equipmentId}", registerGet(getEquipmentStatistics, handleGetEquipmentStatistics)).Methods(http.MethodGet)
	}
}

func handleGetEquipmentStatistics(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseEquipmentId(d.Logger(), func(equipmentId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

			e, err := GetById(d.Logger(), d.Span(), c.Tenant())(equipmentId)
			if err != nil {
				d.Logger().WithError(err).Errorf("Unable to get equipment.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			res, err := model.Transform(e, Transform)
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}
