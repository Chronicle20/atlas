package inventory

import (
	"atlas-character/equipable"
	"atlas-character/equipment/slot"
	"atlas-character/inventory/item"
	"atlas-character/rest"
	"atlas-character/tenant"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/manyminds/api2go/jsonapi"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
	"strconv"
)

const (
	handlerGetItemsForCharacter             = "get_items_for_character"
	handlerRequestGetItemForCharacterByType = "get_item_for_character_by_type"
	handlerGetItemsForCharacterByType       = "get_items_for_character_by_type"
	handlerGetInventoryForCharacterByType   = "get_inventory_for_character_by_type"
	handlerCreateItem                       = "create_item"
	EquipItem                               = "equip_item"
	UnequipItem                             = "unequip_item"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			r := router.PathPrefix("/characters/{characterId}/inventories").Subrouter()
			//r.HandleFunc("/{characterId}/items", registerGetItemsForCharacter(l, db)).Methods(http.MethodGet).Queries("itemId", "{itemId}")
			//r.HandleFunc("/{characterId}/inventories", registerGetItemForCharacterByType(l, db)).Methods(http.MethodGet).Queries("include", "{include}", "type", "{type}", "slot", "{slot}")
			//r.HandleFunc("/{characterId}/inventories", registerGetItemsForCharacterByType(l, db)).Methods(http.MethodGet).Queries("include", "{include}", "type", "{type}", "itemId", "{itemId}")
			//r.HandleFunc("/{characterId}/inventories", registerGetInventoryForCharacterByType(l, db)).Methods(http.MethodGet).Queries("include", "{include}", "type", "{type}")
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

func registerGetInventoryForCharacterByType(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
	return rest.RetrieveSpan(handlerGetInventoryForCharacterByType, func(span opentracing.Span) http.HandlerFunc {
		fl := l.WithFields(logrus.Fields{"originator": handlerGetInventoryForCharacterByType, "type": "rest_handler"})
		return rest.ParseTenant(l, func(tenant tenant.Model) http.HandlerFunc {
			return rest.ParseCharacterId(fl, func(characterId uint32) http.HandlerFunc {
				return handleGetInventoryForCharacterByType(fl, db)(span)(tenant)(characterId)
			})
		})
	})
}

func registerGetItemsForCharacterByType(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
	return rest.RetrieveSpan(handlerGetItemsForCharacterByType, func(span opentracing.Span) http.HandlerFunc {
		fl := l.WithFields(logrus.Fields{"originator": handlerGetItemsForCharacterByType, "type": "rest_handler"})
		return rest.ParseTenant(l, func(tenant tenant.Model) http.HandlerFunc {
			return rest.ParseCharacterId(fl, func(characterId uint32) http.HandlerFunc {
				return handleGetItemsForCharacterByType(fl, db)(span)(tenant)(characterId)
			})
		})
	})
}

func registerGetItemForCharacterByType(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
	return rest.RetrieveSpan(handlerRequestGetItemForCharacterByType, func(span opentracing.Span) http.HandlerFunc {
		fl := l.WithFields(logrus.Fields{"originator": handlerRequestGetItemForCharacterByType, "type": "rest_handler"})
		return rest.ParseTenant(l, func(tenant tenant.Model) http.HandlerFunc {
			return rest.ParseCharacterId(fl, func(characterId uint32) http.HandlerFunc {
				return handleGetItemForCharacterByType(fl, db)(span)(tenant)(characterId)
			})
		})
	})
}

func registerGetItemsForCharacter(l logrus.FieldLogger, db *gorm.DB) http.HandlerFunc {
	return rest.RetrieveSpan(handlerGetItemsForCharacter, func(span opentracing.Span) http.HandlerFunc {
		fl := l.WithFields(logrus.Fields{"originator": handlerGetItemsForCharacter, "type": "rest_handler"})
		return rest.ParseTenant(l, func(tenant tenant.Model) http.HandlerFunc {
			return rest.ParseCharacterId(fl, func(characterId uint32) http.HandlerFunc {
				return handleGetItemsForCharacter(l, db)(span)(tenant)(characterId)
			})
		})
	})
}

func handleGetItemForCharacterByType(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) func(tenant tenant.Model) func(characterId uint32) http.HandlerFunc {
	return func(span opentracing.Span) func(tenant tenant.Model) func(characterId uint32) http.HandlerFunc {
		return func(tenant tenant.Model) func(characterId uint32) http.HandlerFunc {
			return func(characterId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					_ = mux.Vars(r)["include"]

					inventoryType := mux.Vars(r)["type"]
					if inventoryType == "" {
						l.Errorf("Unable to retrieve requested inventory type.")
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					slot, err := strconv.Atoi(mux.Vars(r)["slot"])
					if err != nil {
						l.WithError(err).Errorf("Unable to properly parse slot from path.")
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					_, err = GetInventory(l, db, tenant)(characterId, inventoryType, FilterSlot(int16(slot)))
					if err != nil {
						l.WithError(err).Errorf("Unable to get inventory for character %d by type %s.", characterId, inventoryType)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					//prepareResult(l, db, span, w)(characterId, inv, include)
				}
			}
		}
	}
}

func handleGetItemsForCharacterByType(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) func(tenant tenant.Model) func(characterId uint32) http.HandlerFunc {
	return func(span opentracing.Span) func(tenant tenant.Model) func(characterId uint32) http.HandlerFunc {
		return func(tenant tenant.Model) func(characterId uint32) http.HandlerFunc {
			return func(characterId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					_ = mux.Vars(r)["include"]
					inventoryType := mux.Vars(r)["type"]

					if inventoryType == "" {
						l.Errorf("Unable to retrieve requested inventory type.")
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					itemId, err := strconv.Atoi(mux.Vars(r)["itemId"])
					if err != nil {
						l.WithError(err).Errorf("Unable to properly parse slot from path.")
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					_, err = GetInventory(l, db, tenant)(characterId, inventoryType, FilterItemId(l, db, span, tenant)(uint32(itemId)))
					if err != nil {
						l.WithError(err).Errorf("Unable to get inventory for character %d by type %s.", characterId, inventoryType)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					//prepareResult(l, db, span, w)(characterId, inv, include)
				}
			}
		}
	}
}

func handleGetInventoryForCharacterByType(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) func(tenant tenant.Model) func(characterId uint32) http.HandlerFunc {
	return func(span opentracing.Span) func(tenant tenant.Model) func(characterId uint32) http.HandlerFunc {
		return func(tenant tenant.Model) func(characterId uint32) http.HandlerFunc {
			return func(characterId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					_ = mux.Vars(r)["include"]
					inventoryType := mux.Vars(r)["type"]
					if inventoryType == "" {
						l.Errorf("Unable to retrieve requested inventory type.")
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					_, err := GetInventory(l, db, tenant)(characterId, inventoryType)
					if err != nil {
						l.WithError(err).Errorf("Unable to get inventory for character %d by type %s.", characterId, inventoryType)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					//prepareResult(l, db, span, w)(characterId, inv, include)
				}
			}
		}
	}
}

func handleGetItemsForCharacter(l logrus.FieldLogger, db *gorm.DB) func(span opentracing.Span) func(tenant tenant.Model) func(characterId uint32) http.HandlerFunc {
	return func(span opentracing.Span) func(tenant tenant.Model) func(characterId uint32) http.HandlerFunc {
		return func(tenant tenant.Model) func(characterId uint32) http.HandlerFunc {
			return func(characterId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					itemId, err := strconv.Atoi(mux.Vars(r)["itemId"])
					if err != nil {
						l.WithError(err).Errorf("Unable to properly parse itemId from path.")
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					types := []string{TypeEquip, TypeUse, TypeSetup, TypeETC, TypeCash}
					for _, t := range types {
						_, err := GetInventory(l, db, tenant)(characterId, t, FilterItemId(l, db, span, tenant)(uint32(itemId)))
						if err != nil {
							l.WithError(err).Errorf("Unable to get inventory for character %d by type %s.", characterId, t)
							w.WriteHeader(http.StatusInternalServerError)
							return
						}

						//for _, _ = range inv.Items() {
						//quantity := uint32(1)
						//if val, ok := i.(item.ItemModel); ok {
						//	quantity = val.Quantity()
						//}

						//result.Data = append(result.Data, ItemDataBody{
						//	Id:   strconv.Itoa(int(i.Id())),
						//	Type: getInventoryItemType(inv.Type()),
						//	Attributes: ItemAttributes{
						//		InventoryType: inv.Type(),
						//		Slot:          i.Slot(),
						//		Quantity:      quantity,
						//	},
						//})
						//}
					}

					//w.WriteHeader(http.StatusOK)
					//err = json.ToJSON(result, w)
					//if err != nil {
					//	l.WithError(err).Errorf("Writing response.")
					//}
				}
			}
		}
	}
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
