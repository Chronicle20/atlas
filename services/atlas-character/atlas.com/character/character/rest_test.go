package character_test

import (
	"atlas-character/character"
	"atlas-character/equipable"
	"atlas-character/equipment"
	"atlas-character/equipment/slot"
	"atlas-character/inventory"
	"atlas-character/inventory/item"
	"net/http"
	"net/http/httptest"
	"testing"

	inventory2 "github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/jtumidanski/api2go/jsonapi"
)

type Server struct {
	baseUrl string
	prefix  string
}

func (s Server) GetBaseURL() string {
	return s.baseUrl
}

func (s Server) GetPrefix() string {
	return s.prefix
}

func GetServer() Server {
	return Server{
		baseUrl: "",
		prefix:  "/api/",
	}
}

func TestMarshalUnmarshalSunny(t *testing.T) {
	inv, err := createTestInventory()
	if err != nil {
		t.Fatalf("Failed to create test inventory: %v", err)
	}

	im := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(0).
		SetName("Atlas").
		SetLevel(1).
		SetExperience(0).
		SetInventory(inv).
		SetEquipment(createTestEquipment()).
		Build()

	res, err := model.Map(character.Transform)(model.FixedProvider(im))()
	if err != nil {
		t.Fatalf("Failed to transform model to rest model: %v", err)
	}

	rr := httptest.NewRecorder()
	server.Marshal[character.RestModel](testLogger())(rr)(GetServer())(res)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to write rest model: %v", err)
	}

	body := rr.Body.Bytes()

	output := character.RestModel{}
	err = jsonapi.Unmarshal(body, &output)

	om, err := character.Extract(output)
	if err != nil {
		t.Fatalf("Failed to unmarshal rest model: %v", err)
	}
	if om.Id() != im.Id() {
		t.Fatalf("Failed to unmarshal rest model")
	}

	// do some basic tests
	if im.Id() != om.Id() {
		t.Fatalf("Input and output ids do not match")
	}
	if im.Name() != om.Name() {
		t.Fatalf("Input and output names do not match")
	}
	if !sameEquipment(im, om, "hat") {
		t.Fatalf("Equipment does not match")
	}
	if !sameEquipment(im, om, "weapon") {
		t.Fatalf("Equipment does not match")
	}
	if len(im.GetInventory().Equipable().Items()) != len(om.GetInventory().Equipable().Items()) {
		t.Fatalf("Inventory does not match")
	}
	if len(im.GetInventory().Useable().Items()) != len(om.GetInventory().Useable().Items()) {
		t.Fatalf("Inventory does not match")
	}
	if len(im.GetInventory().Setup().Items()) != len(om.GetInventory().Setup().Items()) {
		t.Fatalf("Inventory does not match")
	}
	if len(im.GetInventory().Etc().Items()) != len(om.GetInventory().Etc().Items()) {
		t.Fatalf("Inventory does not match")
	}
	if len(im.GetInventory().Cash().Items()) != len(om.GetInventory().Cash().Items()) {
		t.Fatalf("Inventory does not match")
	}
}

func sameEquipment(m1 character.Model, m2 character.Model, slotType slot.Type) bool {
	e1, ok := m1.GetEquipment().Get(slotType)
	if !ok {
		return false
	}
	e2, ok := m2.GetEquipment().Get(slotType)
	if !ok {
		return false
	}
	ok = e1.Equipable.Id() == e2.Equipable.Id()
	if !ok {
		return false
	}
	ok = e1.Equipable.Id() == e2.Equipable.Id()
	if !ok {
		return false
	}
	ok = e1.Equipable.ItemId() == e2.Equipable.ItemId()
	if !ok {
		return false
	}
	ok = e1.Equipable.Slot() == e2.Equipable.Slot()
	if !ok {
		return false
	}
	ok = e1.Equipable.ReferenceId() == e2.Equipable.ReferenceId()
	if !ok {
		return false
	}
	ok = e1.Equipable.Strength() == e2.Equipable.Strength()
	if !ok {
		return false
	}
	ok = e1.Equipable.Dexterity() == e2.Equipable.Dexterity()
	if !ok {
		return false
	}
	ok = e1.Equipable.Intelligence() == e2.Equipable.Intelligence()
	if !ok {
		return false
	}
	ok = e1.Equipable.Luck() == e2.Equipable.Luck()
	if !ok {
		return false
	}
	ok = e1.Equipable.HP() == e2.Equipable.HP()
	if !ok {
		return false
	}
	ok = e1.Equipable.MP() == e2.Equipable.MP()
	if !ok {
		return false
	}
	ok = e1.Equipable.WeaponAttack() == e2.Equipable.WeaponAttack()
	if !ok {
		return false
	}
	ok = e1.Equipable.MagicAttack() == e2.Equipable.MagicAttack()
	if !ok {
		return false
	}
	ok = e1.Equipable.WeaponDefense() == e2.Equipable.WeaponDefense()
	if !ok {
		return false
	}
	ok = e1.Equipable.MagicDefense() == e2.Equipable.MagicDefense()
	if !ok {
		return false
	}
	ok = e1.Equipable.Accuracy() == e2.Equipable.Accuracy()
	if !ok {
		return false
	}
	ok = e1.Equipable.Avoidability() == e2.Equipable.Avoidability()
	if !ok {
		return false
	}
	ok = e1.Equipable.Hands() == e2.Equipable.Hands()
	if !ok {
		return false
	}
	ok = e1.Equipable.Speed() == e2.Equipable.Speed()
	if !ok {
		return false
	}
	ok = e1.Equipable.Jump() == e2.Equipable.Jump()
	if !ok {
		return false
	}
	ok = e1.Equipable.Slots() == e2.Equipable.Slots()
	return ok
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
	usm, err := model.Fold(model.FixedProvider(useItems), inventory.NewItemModel(inv.Useable().Id(), inventory2.TypeValueUse, inv.Useable().Capacity()), inventory.ItemFolder)()
	if err != nil {
		return inventory.Model{}, err
	}

	setupItems := []item.Model{
		item.NewModelBuilder().SetID(8).SetItemId(3010046).SetSlot(2).SetQuantity(1).Build(),
		item.NewModelBuilder().SetID(9).SetItemId(3010003).SetSlot(1).SetQuantity(1).Build(),
	}
	sem, err := model.Fold(model.FixedProvider(setupItems), inventory.NewItemModel(inv.Setup().Id(), inventory2.TypeValueSetup, inv.Setup().Capacity()), inventory.ItemFolder)()
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
	etm, err := model.Fold(model.FixedProvider(etcItems), inventory.NewItemModel(inv.Etc().Id(), inventory2.TypeValueETC, inv.Etc().Capacity()), inventory.ItemFolder)()
	if err != nil {
		return inventory.Model{}, err
	}

	cashItems := []item.Model{
		item.NewModelBuilder().SetID(53).SetItemId(5000020).SetSlot(2).SetQuantity(1).Build(),
		item.NewModelBuilder().SetID(51).SetItemId(5370000).SetSlot(1).SetQuantity(1).Build(),
	}
	cam, err := model.Fold(model.FixedProvider(cashItems), inventory.NewItemModel(inv.Cash().Id(), inventory2.TypeValueCash, inv.Cash().Capacity()), inventory.ItemFolder)()
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

	eqm := equipment.NewModel()
	eqm.SetEquipable("hat", false, hat)
	eqm.SetEquipable("top", false, top)
	eqm.SetEquipable("weapon", false, weapon)
	eqm.SetEquipable("pants", false, bottoms)
	eqm.SetEquipable("shoes", false, shoes)
	eqm.SetEquipable("shoes", true, cashShoes)
	return eqm
}
