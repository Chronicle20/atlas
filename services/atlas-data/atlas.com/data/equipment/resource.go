package equipment

import (
	"atlas-data/rest"
	"errors"
	"net/http"
	"sort"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate"
)

func InitResource(db *gorm.DB) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(si)

			r := router.PathPrefix("/data/equipment").Subrouter()
			r.HandleFunc("/{equipmentId}", registerGet("get_equipment_statistics", handleGetEquipmentStatistics(db))).Methods(http.MethodGet)
			r.HandleFunc("/{equipmentId}/slots", registerGet("get_equipment_slots", handleGetEquipmentSlots(db))).Methods(http.MethodGet)
		}
	}
}

func handleGetEquipmentStatistics(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseEquipmentId(d.Logger(), func(equipmentId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				s := NewStorage(d.Logger(), db)
				res, err := s.GetById(d.Context())(strconv.Itoa(int(equipmentId)))
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						d.Logger().WithError(err).Warnf("Equipment template [%d] not found; seed data may be missing.", equipmentId)
						w.WriteHeader(http.StatusNotFound)
						return
					}
					d.Logger().WithError(err).Errorf("Unable to get equipment [%d].", equipmentId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}
				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}

func handleGetEquipmentSlots(db *gorm.DB) func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseEquipmentId(d.Logger(), func(equipmentId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				query := r.URL.Query()
				page, err := paginate.ParseParams(query, paginate.DefaultPageSize, paginate.MaxPageSize)
				if err != nil {
					server.WriteBadRequest(d.Logger(), w, err.Error())
					return
				}

				s := NewStorage(d.Logger(), db)
				res, err := s.GetById(d.Context())(strconv.Itoa(int(equipmentId)))
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						d.Logger().WithError(err).Warnf("Equipment template [%d] not found; seed data may be missing.", equipmentId)
						w.WriteHeader(http.StatusNotFound)
						return
					}
					d.Logger().WithError(err).Errorf("Unable to get equipment [%d].", equipmentId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				slots := res.EquipSlots
				sort.SliceStable(slots, func(i, j int) bool {
					return slots[i].Slot < slots[j].Slot
				})

				paged := paginate.Slice(slots, page)
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalPaginatedResponse[[]SlotRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(paged.Items, paginate.EnvelopeFor(paged), r)
			}
		})
	}
}
