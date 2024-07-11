package inventory

import (
	"atlas-character/equipable"
	"atlas-character/equipment/slot"
	"atlas-character/inventory/item"
	"atlas-character/rest"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

const (
	handlerCreateItem = "create_item"
	EquipItem         = "equip_item"
	UnequipItem       = "unequip_item"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/characters/{characterId}/inventories").Subrouter()
			r.HandleFunc("/{inventoryType}/items", rest.RegisterCreateHandler[item.RestModel](l)(db)(si)(handlerCreateItem, handleCreateItem)).Methods(http.MethodPost)

			register := rest.RegisterHandler(l)(db)(si)
			registerCreate := rest.RegisterCreateHandler[equipable.RestModel](l)(db)(si)
			er := router.PathPrefix("/characters/{characterId}/equipment").Subrouter()

			er.HandleFunc("/{slotType}/equipable", registerCreate(EquipItem, handleEquipItem)).Methods(http.MethodPost)
			er.HandleFunc("/{slotType}/equipable", register(UnequipItem, handleUnequipItem)).Methods(http.MethodDelete)
		}
	}
}

func handleCreateItem(d *rest.HandlerDependency, c *rest.HandlerContext, model item.RestModel) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return rest.ParseInventoryType(d.Logger(), func(inventoryType int8) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := CreateItem(d.Logger(), d.DB(), d.Span(), c.Tenant())(characterId, Type(inventoryType), model.ItemId, model.Quantity)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusAccepted)
				return
			}
		})
	})
}

type SlotTypeHandler func(slotType string) http.HandlerFunc

func ParseSlotType(l logrus.FieldLogger, next SlotTypeHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if val, ok := mux.Vars(r)["slotType"]; ok {
			next(val)(w, r)
			return
		}
		l.Errorf("Unable to properly parse slotType from path.")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func handleEquipItem(d *rest.HandlerDependency, c *rest.HandlerContext, input equipable.RestModel) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return ParseSlotType(d.Logger(), func(slotType string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				des, err := slot.PositionFromType(slotType)
				if err != nil {
					d.Logger().Errorf("Slot type [%s] does not map to a valid equipment position.", slotType)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				emitEquipItemCommand(d.Logger(), d.Span(), c.Tenant())(characterId, input.Slot, int16(des))
				w.WriteHeader(http.StatusAccepted)
			}
		})
	})
}

func handleUnequipItem(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return ParseSlotType(d.Logger(), func(slotType string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				des, err := slot.PositionFromType(slotType)
				if err != nil {
					d.Logger().Errorf("Slot type [%s] does not map to a valid equipment position.", slotType)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				emitUnequipItemCommand(d.Logger(), d.Span(), c.Tenant())(characterId, int16(des))
				w.WriteHeader(http.StatusAccepted)
			}
		})
	})
}
