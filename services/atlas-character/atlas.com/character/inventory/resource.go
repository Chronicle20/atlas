package inventory

import (
	"atlas-character/equipable"
	"atlas-character/equipment/slot"
	"atlas-character/inventory/item"
	"atlas-character/kafka/producer"
	"atlas-character/rest"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
	"strconv"
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
			r.HandleFunc("/{inventoryType}/items", rest.RegisterCreateHandler[item.RestModel](l)(db)(si)(handlerCreateItem, handleCreateItem)).Methods(http.MethodPost)
			r.HandleFunc("/{inventoryType}/items", register(getItemBySlot, handleGetItemBySlot)).Methods(http.MethodGet).Queries("slot", "{slot}")

			er := router.PathPrefix("/characters/{characterId}/equipment").Subrouter()
			er.HandleFunc("/{slotType}/equipable", rest.RegisterCreateHandler[equipable.RestModel](l)(db)(si)(EquipItem, handleEquipItem)).Methods(http.MethodPost)
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

				inv, err := GetInventories(d.Logger(), d.DB(), d.Span(), c.Tenant())(characterId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Unable to get inventory for character [%d].", characterId)
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				if Type(inventoryType) == TypeValueEquip {
					for _, i := range inv.Equipable().Items() {
						if i.Slot() == int16(slot) {
							res, err := model.Map(model.FixedProvider(i), equipable.Transform)()
							if err != nil {
								d.Logger().WithError(err).Errorf("Creating REST model.")
								w.WriteHeader(http.StatusInternalServerError)
								return
							}

							server.Marshal[equipable.RestModel](d.Logger())(w)(c.ServerInformation())(res)
							return
						}
					}
					w.WriteHeader(http.StatusNotFound)
					return
				}

				var m ItemModel
				switch Type(inventoryType) {
				case TypeValueUse:
					m = inv.Useable()
				case TypeValueSetup:
					m = inv.Setup()
				case TypeValueETC:
					m = inv.Etc()
				case TypeValueCash:
					m = inv.Cash()
				default:
					d.Logger().WithError(err).Errorf("Unable to get inventory for character [%d].", characterId)
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				for _, i := range m.Items() {
					if i.Slot() == int16(slot) {
						res, err := model.Map(model.FixedProvider(i), item.Transform)()
						if err != nil {
							d.Logger().WithError(err).Errorf("Creating REST model.")
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						server.Marshal[item.RestModel](d.Logger())(w)(c.ServerInformation())(res)
						return
					}
				}
				w.WriteHeader(http.StatusNotFound)
				return
			}
		})
	})
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
				_ = producer.ProviderImpl(d.Logger())(d.Span())(EnvCommandTopicEquipItem)(equipItemCommandProvider(c.Tenant(), characterId, input.Slot, int16(des)))
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
				_ = producer.ProviderImpl(d.Logger())(d.Span())(EnvCommandTopicUnequipItem)(unequipItemCommandProvider(c.Tenant(), characterId, int16(des)))
				w.WriteHeader(http.StatusAccepted)
			}
		})
	})
}
