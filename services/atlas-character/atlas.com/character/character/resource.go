package character

import (
	"atlas-character/equipable"
	"atlas-character/equipment"
	"atlas-character/inventory"
	"atlas-character/inventory/item"
	"atlas-character/kafka/producer"
	"atlas-character/rest"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"strings"
)

func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			r := router.PathPrefix("/characters").Subrouter()
			r.HandleFunc("", registerGet("get_characters_for_account_in_world", handleGetCharactersForAccountInWorld)).Methods(http.MethodGet).Queries("accountId", "{accountId}", "worldId", "{worldId}", "include", "{include}")
			r.HandleFunc("", registerGet("get_characters_for_account_in_world", handleGetCharactersForAccountInWorld)).Methods(http.MethodGet).Queries("accountId", "{accountId}", "worldId", "{worldId}")
			r.HandleFunc("", registerGet("get_characters_by_map", handleGetCharactersByMap)).Methods(http.MethodGet).Queries("worldId", "{worldId}", "mapId", "{mapId}", "include", "{include}")
			r.HandleFunc("", registerGet("get_characters_by_map", handleGetCharactersByMap)).Methods(http.MethodGet).Queries("worldId", "{worldId}", "mapId", "{mapId}")
			r.HandleFunc("", registerGet("get_characters_by_name", handleGetCharactersByName)).Methods(http.MethodGet).Queries("name", "{name}", "include", "{include}")
			r.HandleFunc("", registerGet("get_characters_by_name", handleGetCharactersByName)).Methods(http.MethodGet).Queries("name", "{name}")
			r.HandleFunc("", registerGet("get_characters", handleGetCharacters)).Methods(http.MethodGet)
			r.HandleFunc("", rest.RegisterInputHandler[RestModel](l)(db)(si)("create_character", handleCreateCharacter)).Methods(http.MethodPost)
			r.HandleFunc("/{characterId}", registerGet("get_character", handleGetCharacter)).Methods(http.MethodGet).Queries("include", "{include}")
			r.HandleFunc("/{characterId}", registerGet("get_character", handleGetCharacter)).Methods(http.MethodGet)
			r.HandleFunc("/{characterId}", rest.RegisterHandler(l)(db)(si)("delete_character", handleDeleteCharacter)).Methods(http.MethodDelete)
		}
	}
}

func handleGetCharacters(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cs, err := GetAll(d.DB())(d.Context())(decoratorsFromInclude(r, d, c)...)
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to get characters.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.SliceMap(Transform)(model.FixedProvider(cs))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[[]RestModel](d.Logger())(w)(c.ServerInformation())(res)
	}
}

func handleGetCharactersForAccountInWorld(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accountId, err := strconv.Atoi(mux.Vars(r)["accountId"])
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to properly parse accountId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		worldId, err := strconv.Atoi(mux.Vars(r)["worldId"])
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to properly parse worldId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		cs, err := GetForAccountInWorld(d.DB())(d.Context())(uint32(accountId), byte(worldId), decoratorsFromInclude(r, d, c)...)
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to get characters for account %d in world %d.", accountId, worldId)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.SliceMap(Transform)(model.FixedProvider(cs))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[[]RestModel](d.Logger())(w)(c.ServerInformation())(res)
	}
}

func decoratorsFromInclude(r *http.Request, d *rest.HandlerDependency, _ *rest.HandlerContext) []model.Decorator[Model] {
	var decorators = make([]model.Decorator[Model], 0)
	include := mux.Vars(r)["include"]
	if strings.Contains(include, "inventory") {
		decorators = append(decorators, InventoryModelDecorator(d.Logger())(d.DB())(d.Context()))
	}
	return decorators
}

func handleGetCharactersByMap(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		worldId, err := strconv.Atoi(mux.Vars(r)["worldId"])
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to properly parse worldId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mapId, err := strconv.Atoi(mux.Vars(r)["mapId"])
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to properly parse mapId from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		cs, err := GetForMapInWorld(d.DB())(d.Context())(byte(worldId), uint32(mapId), decoratorsFromInclude(r, d, c)...)
		if err != nil {
			d.Logger().WithError(err).Errorf("Unable to get characters for map %d in world %d.", mapId, worldId)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.SliceMap(Transform)(model.FixedProvider(cs))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[[]RestModel](d.Logger())(w)(c.ServerInformation())(res)
	}
}

func handleGetCharactersByName(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name, ok := mux.Vars(r)["name"]
		if !ok {
			d.Logger().Errorf("Unable to properly parse name from path.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		cs, err := GetForName(d.DB())(d.Context())(name, decoratorsFromInclude(r, d, c)...)
		if err != nil {
			d.Logger().WithError(err).Errorf("Getting character %s.", name)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.SliceMap(Transform)(model.FixedProvider(cs))(model.ParallelMap())()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[[]RestModel](d.Logger())(w)(c.ServerInformation())(res)
	}
}

func createTestInventory() (inventory.Model, error) {
	inv := inventory.NewModel(24)

	var equipables = []equipable.Model{
		equipable.NewModelBuilder().SetID(33).SetItemId(1002067).SetSlot(3).SetWeaponDefense(5).SetSlots(7).Build(),
		equipable.NewModelBuilder().SetID(34).SetItemId(1040002).SetSlot(4).SetWeaponDefense(3).SetSlots(7).Build(),
		equipable.NewModelBuilder().SetID(8).SetItemId(1322013).SetSlot(2).SetWeaponAttack(200).SetSlots(7).Build(),
		equipable.NewModelBuilder().SetID(10).SetItemId(1072153).SetSlot(1).Build(),
		equipable.NewModelBuilder().SetID(35).SetItemId(1302000).SetSlot(5).SetWeaponAttack(17).SetSlots(7).Build(),
	}
	em, err := model.Fold(model.FixedProvider(equipables), inventory.NewEquipableModel(inv.Equipable().Id(), inv.Equipable().Capacity()), inventory.EquipableFolder)()
	if err != nil {
		return inventory.Model{}, err
	}

	useItems := []item.Model{
		item.NewModelBuilder().SetID(22).SetItemId(2070006).SetSlot(1).SetQuantity(100).Build(),
		item.NewModelBuilder().SetID(25).SetItemId(2041043).SetSlot(4).SetQuantity(1).Build(),
		item.NewModelBuilder().SetID(26).SetItemId(2041045).SetSlot(5).SetQuantity(1).Build(),
		item.NewModelBuilder().SetID(29).SetItemId(2380000).SetSlot(8).SetQuantity(1).Build(),
		item.NewModelBuilder().SetID(28).SetItemId(2061000).SetSlot(7).SetQuantity(3).Build(),
		item.NewModelBuilder().SetID(27).SetItemId(2060000).SetSlot(6).SetQuantity(3).Build(),
		item.NewModelBuilder().SetID(23).SetItemId(2000000).SetSlot(2).SetQuantity(3).Build(),
		item.NewModelBuilder().SetID(24).SetItemId(2010009).SetSlot(3).SetQuantity(5).Build(),
	}
	usm, err := model.Fold(model.FixedProvider(useItems), inventory.NewItemModel(inv.Useable().Id(), inventory.TypeValueUse, inv.Useable().Capacity()), inventory.ItemFolder)()
	if err != nil {
		return inventory.Model{}, err
	}

	setupItems := []item.Model{
		item.NewModelBuilder().SetID(8).SetItemId(3010046).SetSlot(2).SetQuantity(1).Build(),
		item.NewModelBuilder().SetID(9).SetItemId(3010003).SetSlot(1).SetQuantity(1).Build(),
	}
	sem, err := model.Fold(model.FixedProvider(setupItems), inventory.NewItemModel(inv.Setup().Id(), inventory.TypeValueSetup, inv.Setup().Capacity()), inventory.ItemFolder)()
	if err != nil {
		return inventory.Model{}, err
	}

	etcItems := []item.Model{
		item.NewModelBuilder().SetID(1).SetItemId(4161001).SetSlot(1).SetQuantity(1).Build(),
		item.NewModelBuilder().SetID(5).SetItemId(4000016).SetSlot(2).SetQuantity(85).Build(),
		item.NewModelBuilder().SetID(31).SetItemId(4010000).SetSlot(5).SetQuantity(3).Build(),
		item.NewModelBuilder().SetID(12).SetItemId(4000002).SetSlot(3).SetQuantity(12).Build(),
		item.NewModelBuilder().SetID(32).SetItemId(4020000).SetSlot(6).SetQuantity(1).Build(),
		item.NewModelBuilder().SetID(33).SetItemId(4000008).SetSlot(7).SetQuantity(2).Build(),
		item.NewModelBuilder().SetID(34).SetItemId(4000176).SetSlot(8).SetQuantity(1).Build(),
		item.NewModelBuilder().SetID(35).SetItemId(4000000).SetSlot(9).SetQuantity(1).Build(),
		item.NewModelBuilder().SetID(30).SetItemId(4000019).SetSlot(4).SetQuantity(29).Build(),
		item.NewModelBuilder().SetID(50).SetItemId(4006001).SetSlot(10).SetQuantity(100).Build(),
	}
	etm, err := model.Fold(model.FixedProvider(etcItems), inventory.NewItemModel(inv.Etc().Id(), inventory.TypeValueETC, inv.Etc().Capacity()), inventory.ItemFolder)()
	if err != nil {
		return inventory.Model{}, err
	}

	cashItems := []item.Model{
		item.NewModelBuilder().SetID(53).SetItemId(5000020).SetSlot(2).SetQuantity(1).Build(),
		item.NewModelBuilder().SetID(51).SetItemId(5370000).SetSlot(1).SetQuantity(1).Build(),
	}
	cam, err := model.Fold(model.FixedProvider(cashItems), inventory.NewItemModel(inv.Cash().Id(), inventory.TypeValueCash, inv.Cash().Capacity()), inventory.ItemFolder)()
	if err != nil {
		return inventory.Model{}, err
	}

	inv = inv.SetEquipable(em).
		SetUseable(usm).
		SetSetup(sem).
		SetEtc(etm).
		SetCash(cam)
	return inv, err
}

func createTestEquipment() equipment.Model {
	hat := equipable.NewModelBuilder().
		SetID(5).
		SetItemId(1002140).
		SetSlot(-1).
		SetStrength(999).
		SetDexterity(999).
		SetIntelligence(999).
		SetLuck(999).
		SetWeaponDefense(200).
		SetMagicDefense(200).
		SetSpeed(30).
		SetJump(50).
		SetSlots(7).
		Build()
	top := equipable.NewModelBuilder().
		SetID(6).
		SetItemId(1042003).
		SetSlot(-5).
		Build()
	weapon := equipable.NewModelBuilder().
		SetID(24).
		SetItemId(1472000).
		SetSlot(-11).
		SetWeaponAttack(10).
		SetSlots(7).
		Build()
	bottoms := equipable.NewModelBuilder().
		SetID(7).
		SetItemId(1062007).
		SetSlot(-6).
		Build()
	shoes := equipable.NewModelBuilder().
		SetID(1).
		SetItemId(1072037).
		SetSlot(-7).
		SetWeaponDefense(2).
		SetSlots(5).
		Build()
	cashShoes := equipable.NewModelBuilder().
		SetID(9).
		SetItemId(1070005).
		SetSlot(-107).
		Build()

	eqm := equipment.NewModel().
		SetHat(&hat).
		SetTop(&top).
		SetWeapon(&weapon).
		SetBottom(&bottoms).
		SetShoes(&shoes).
		SetCashShoes(&cashShoes)
	return eqm
}

func handleGetCharacter(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			//cs, err := GetById(d.Context())(d.DB())(decoratorsFromInclude(r, d, c)...)(characterId)
			//if errors.Is(err, gorm.ErrRecordNotFound) {
			//	w.WriteHeader(http.StatusNotFound)
			//	return
			//}

			inv, err := createTestInventory()
			if err != nil {
				return
			}

			im := NewModelBuilder().
				SetAccountId(1000).
				SetWorldId(0).
				SetName("Atlas").
				SetLevel(1).
				SetExperience(0).
				SetInventory(inv).
				SetEquipment(createTestEquipment()).
				Build()

			res, err := model.Map(Transform)(model.FixedProvider(im))()
			if err != nil {
				d.Logger().WithError(err).Errorf("Creating REST model.")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
		}
	})
}

func handleCreateCharacter(d *rest.HandlerDependency, c *rest.HandlerContext, input RestModel) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m, err := Extract(input)
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		cs, err := Create(d.Logger())(d.DB())(d.Context())(producer.ProviderImpl(d.Logger())(d.Context()))(m)
		if err != nil {
			if errors.Is(err, blockedNameErr) || errors.Is(err, invalidLevelErr) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			d.Logger().WithError(err).Errorf("Creating character.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		res, err := model.Map(Transform)(model.FixedProvider(cs))()
		if err != nil {
			d.Logger().WithError(err).Errorf("Creating REST model.")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		server.Marshal[RestModel](d.Logger())(w)(c.ServerInformation())(res)
	}
}

func handleDeleteCharacter(d *rest.HandlerDependency, _ *rest.HandlerContext) http.HandlerFunc {
	return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := Delete(d.Logger())(d.DB())(d.Context())(producer.ProviderImpl(d.Logger())(d.Context()))(characterId)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}
	})
}
