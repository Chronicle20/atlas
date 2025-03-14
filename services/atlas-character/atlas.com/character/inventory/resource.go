package inventory

import (
	"atlas-character/equipable"
	"atlas-character/equipment/slot"
	"atlas-character/inventory/item"
	"atlas-character/kafka/producer"
	"atlas-character/rest"
	"net/http"
	"strconv"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	handlerCreateItem = "create_item"
	EquipItem         = "equip_item"
	UnequipItem       = "unequip_item"
	getItemBySlot     = "get_item_by_slot"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			register := rest.RegisterHandler(l)(db)(si)

			r := router.PathPrefix("/characters/{characterId}/inventories").Subrouter()
			r.HandleFunc("/{inventoryType}/items", rest.RegisterInputHandler[item.RestModel](l)(db)(si)(handlerCreateItem, handleCreateItem)).Methods(http.MethodPost)
			r.HandleFunc("/{inventoryType}/items", register(getItemBySlot, handleGetItemBySlot)).Methods(http.MethodGet).Queries("slot", "{slot}")

			er := router.PathPrefix("/characters/{characterId}/equipment").Subrouter()
			er.HandleFunc("/{slotType}/equipable", rest.RegisterInputHandler[equipable.RestModel](l)(db)(si)(EquipItem, handleEquipItem)).Methods(http.MethodPost)
			er.HandleFunc("/{slotType}/equipable", register(UnequipItem, handleUnequipItem)).Methods(http.MethodDelete)
		}
	}
}

func handleGetItemBySlot(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return rest.ParseInventoryType(d.Logger(), func(inventoryType int8) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				slot, err := strconv.Atoi(mux.Vars(r)["slot"])
				if err != nil {
					d.Logger().Errorf("Unable to properly parse slot from path.")
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				inv, err := GetInventories(d.Logger())(d.DB())(d.Context())(characterId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to get inventory for character [%d].", characterId)
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				if inventory.Type(inventoryType) == inventory.TypeValueEquip {
					for _, i := range inv.Equipable().Items() {
						if i.Slot() == int16(slot) {
							res, err := model.Map(equipable.Transform)(model.FixedProvider(i))()
							if err != nil {
								d.Logger().WithError(err).Errorf("Creating REST model.")
								w.WriteHeader(http.StatusInternalServerError)
								return
							}

							query := r.URL.Query()
							queryParams := jsonapi.ParseQueryFields(&query)
							server.MarshalResponse[equipable.RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
							return
						}
					}
					w.WriteHeader(http.StatusNotFound)
					return
				}

				var m ItemModel
				switch inventory.Type(inventoryType) {
				case inventory.TypeValueUse:
					m = inv.Useable()
				case inventory.TypeValueSetup:
					m = inv.Setup()
				case inventory.TypeValueETC:
					m = inv.Etc()
				case inventory.TypeValueCash:
					m = inv.Cash()
				default:
					d.Logger().WithError(err).Errorf("Unable to get inventory for character [%d].", characterId)
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				for _, i := range m.Items() {
					if i.Slot() == int16(slot) {
						res, err := model.Map(item.Transform)(model.FixedProvider(i))()
						if err != nil {
							d.Logger().WithError(err).Errorf("Creating REST model.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						query := r.URL.Query()
						queryParams := jsonapi.ParseQueryFields(&query)
						server.MarshalResponse[item.RestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
						return
					}
				}
				w.WriteHeader(http.StatusNotFound)
			}
		})
	})
}

func handleCreateItem(d *rest.HandlerDependency, _ *rest.HandlerContext, model item.RestModel) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return rest.ParseInventoryType(d.Logger(), func(inventoryType int8) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				err := CreateItem(d.Logger())(d.DB())(d.Context())(producer.ProviderImpl(d.Logger())(d.Context()))(characterId, inventory.Type(inventoryType), model.ItemId, model.Quantity)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusAccepted)
			}
		})
	})
}

type SlotTypeHandler func(slotType slot.Type) http.HandlerFunc

func ParseSlotType(l logrus.FieldLogger, next SlotTypeHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if val, ok := mux.Vars(r)["slotType"]; ok {
			next(slot.Type(val))(w, r)
			return
		}
		l.Errorf("Unable to properly parse slotType from path.")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func handleEquipItem(d *rest.HandlerDependency, c *rest.HandlerContext, input equipable.RestModel) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return ParseSlotType(d.Logger(), func(slotType slot.Type) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				des, err := slot.GetSlotByType(slotType)
				if err != nil {
					d.Logger().Errorf("Slot type [%s] does not map to a valid equipment position.", slotType)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				_ = producer.ProviderImpl(d.Logger())(d.Context())(EnvCommandTopic)(equipItemCommandProvider(characterId, input.Slot, int16(des.Position)))
				w.WriteHeader(http.StatusAccepted)
			}
		})
	})
}

func handleUnequipItem(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return ParseSlotType(d.Logger(), func(slotType slot.Type) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				des, err := slot.GetSlotByType(slotType)
				if err != nil {
					d.Logger().Errorf("Slot type [%s] does not map to a valid equipment position.", slotType)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				_ = producer.ProviderImpl(d.Logger())(d.Context())(EnvCommandTopic)(unequipItemCommandProvider(characterId, int16(des.Position), 0))
				w.WriteHeader(http.StatusAccepted)
			}
		})
	})
}
