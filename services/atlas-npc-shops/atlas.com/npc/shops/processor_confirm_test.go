package shops

import (
	"atlas-npc/asset"
	"atlas-npc/character"
	"atlas-npc/commodities"
	"atlas-npc/compartment"
	inventory2 "atlas-npc/inventory"
	"atlas-npc/kafka/message"
	characterMessage "atlas-npc/kafka/message/character"
	compartmentMessage "atlas-npc/kafka/message/compartment"
	"atlas-npc/kafka/message/shops"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// These tests pin the contract the client enforces on itself: CShopDlg latches
// a "request outstanding" flag at CShopDlg+0xFC when it sends a buy, sell or
// recharge (CShopDlg::SendSellRequest @0x756a04, SendRechargeRequest
// @0x756c28) and CShopDlg::OnPacket @0x756da7 clears it only on receipt of
// CONFIRM_SHOP_TRANSACTION. Every terminal outcome of a shop request — success
// as much as refusal — must publish exactly one status event, or the dialog
// stays wedged until the player closes and reopens it.

const (
	// 2000000 is a plain USE consumable: not rechargeable, so Buy takes the
	// meso path without consulting the data service.
	testPotionId = uint32(2000000)
	// 2070000 (Subi throwing-stars) classifies as a throwing star, which is
	// what makes it rechargeable.
	testStarId = uint32(2070000)
	// 4000000 is an ETC item, priced through the data service on sell.
	testEtcId  = uint32(4000000)
	testShopId = uint32(9000069)
)

// stubCommodities serves a fixed commodity list. Only GetByNpcId is exercised
// here; the embedded interface leaves the rest unimplemented on purpose, so an
// unexpected call panics loudly rather than returning a silent zero value.
type stubCommodities struct {
	commodities.Processor
	cms []commodities.Model
}

func (s stubCommodities) GetByNpcId(_ uint32) ([]commodities.Model, error) {
	return s.cms, nil
}

// stubCharacter serves a fixed character. Meso movement is delegated to the
// production provider so the command shape stays honest.
type stubCharacter struct {
	character.Processor
	m character.Model
}

func (s stubCharacter) GetById(_ ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error) {
	return func(_ uint32) (character.Model, error) {
		return s.m, nil
	}
}

func (s stubCharacter) InventoryDecorator(m character.Model) character.Model {
	return m
}

func (s stubCharacter) RequestChangeMeso(mb *message.Buffer) func(worldId world.Id, characterId uint32, actorId uint32, actorType string, amount int32) error {
	return func(worldId world.Id, characterId uint32, actorId uint32, actorType string, amount int32) error {
		return mb.Put(characterMessage.EnvCommandTopic,
			character.RequestChangeMesoCommandProvider(characterId, worldId, actorId, actorType, amount))
	}
}

func confirmProcessor(t *testing.T, ctx context.Context, db *gorm.DB, cp commodities.Processor, c character.Model) *ProcessorImpl {
	t.Helper()
	l := logrus.New()
	l.SetOutput(io.Discard)
	return &ProcessorImpl{
		l:     l,
		ctx:   ctx,
		db:    db,
		t:     tenant.MustFromContext(ctx),
		cp:    cp,
		charP: stubCharacter{m: c},
		compP: compartment.NewProcessor(),
	}
}

func confirmTestCharacter(t *testing.T, meso uint32, comps ...compartment.Model) character.Model {
	t.Helper()
	ib := inventory2.NewBuilder(testCharacterId)
	for _, c := range comps {
		ib = ib.SetCompartment(c)
	}
	return character.NewBuilder().
		SetId(testCharacterId).
		SetMeso(meso).
		SetInventory(ib.Build()).
		Build()
}

func useCompartment(assets ...asset.Model) compartment.Model {
	return compartment.NewBuilder(uuid.New(), testCharacterId, inventory.TypeValueUse, 24).
		SetAssets(assets).
		Build()
}

func etcCompartment(assets ...asset.Model) compartment.Model {
	return compartment.NewBuilder(uuid.New(), testCharacterId, inventory.TypeValueETC, 24).
		SetAssets(assets).
		Build()
}

type consumableDoc struct {
	SlotMax   uint32  `json:"slotMax"`
	UnitPrice float64 `json:"unitPrice"`
	Price     uint32  `json:"price"`
}

type etcDoc struct {
	Price uint32 `json:"price"`
}

// dataServer stands in for the data and skills services. Templates are served
// as single JSON:API documents; the skills list is served as an empty
// paginated envelope, which is the shape DrainProvider expects.
func dataServer(t *testing.T, consumables map[uint32]consumableDoc, etcs map[uint32]etcDoc) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/skills"):
			_, _ = w.Write([]byte(`{"data":[],"meta":{"total":0,"page":{"number":1,"size":10,"last":1}}}`))
		case strings.Contains(path, "/data/consumables/"):
			doc, ok := consumables[trailingId(t, path)]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			writeDoc(t, w, "consumables", trailingId(t, path), doc)
		case strings.Contains(path, "/data/etcs/"):
			doc, ok := etcs[trailingId(t, path)]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			writeDoc(t, w, "etcs", trailingId(t, path), doc)
		default:
			t.Errorf("unexpected request to %s", path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")
	t.Setenv("SKILLS_SERVICE_URL", srv.URL+"/")
}

func trailingId(t *testing.T, path string) uint32 {
	t.Helper()
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	id, err := strconv.ParseUint(parts[len(parts)-1], 10, 32)
	if err != nil {
		t.Fatalf("could not parse an id out of %q: %v", path, err)
	}
	return uint32(id)
}

func writeDoc(t *testing.T, w io.Writer, resourceType string, id uint32, attributes any) {
	t.Helper()
	attrs, err := json.Marshal(attributes)
	if err != nil {
		t.Fatalf("failed to marshal attributes: %v", err)
	}
	body := fmt.Sprintf(`{"data":{"type":%q,"id":"%d","attributes":%s}}`, resourceType, id, attrs)
	if _, err = w.Write([]byte(body)); err != nil {
		t.Fatalf("failed writing response: %v", err)
	}
}

func confirmTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	l := logrus.New()
	l.SetOutput(io.Discard)
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	database.RegisterTenantCallbacks(l, db)
	if err = Migration(db); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func seedRechargerShop(t *testing.T, db *gorm.DB, ctx context.Context) {
	t.Helper()
	ten := tenant.MustFromContext(ctx)
	e := Entity{
		Id:        uuid.New(),
		TenantId:  ten.Id(),
		NpcId:     testShopId,
		Recharger: true,
	}
	if err := db.Create(&e).Error; err != nil {
		t.Fatalf("failed to seed shop: %v", err)
	}
}

// assertSingleOk is the assertion these tests exist for: exactly one status
// event, and it is the OK arm that unlatches the client.
func assertSingleOk(t *testing.T, buf *message.Buffer) {
	t.Helper()
	events := buf.GetAll()[shops.EnvStatusEventTopic]
	if len(events) != 1 {
		t.Fatalf("expected exactly one status event, got %d", len(events))
	}
	ev := decodeStatusError(t, events[0].Value)
	if ev.Type != shops.StatusEventTypeError {
		t.Errorf("status event type: got %q want %q", ev.Type, shops.StatusEventTypeError)
	}
	if ev.CharacterId != testCharacterId {
		t.Errorf("status event characterId: got %d want %d", ev.CharacterId, testCharacterId)
	}
	if ev.Body.Error != shops.ErrorOk {
		t.Errorf("status event error: got %q want %q", ev.Body.Error, shops.ErrorOk)
	}
}

func mesoCommodity(t *testing.T, templateId uint32, price uint32) commodities.Model {
	t.Helper()
	cm, err := commodities.NewBuilder().
		SetId(uuid.New()).
		SetNpcId(testShopId).
		SetTemplateId(templateId).
		SetMesoPrice(price).
		Build()
	if err != nil {
		t.Fatalf("failed to build commodity: %v", err)
	}
	return cm
}

func enterShop(t *testing.T) context.Context {
	t.Helper()
	setupTestRegistry(t)
	ctx := testTenantCtx()
	GetRegistry().AddCharacter(ctx, testCharacterId, testShopId)
	return ctx
}

// The channel consumer switches on Type == ERROR and then on Body.Error, so
// the OK arm must travel as an ERROR-typed event carrying "OK".
func TestOkEventProviderMatchesConsumerSwitch(t *testing.T) {
	msgs, err := okEventProvider(testCharacterId)()
	if err != nil {
		t.Fatalf("okEventProvider returned an error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected one message, got %d", len(msgs))
	}
	ev := decodeStatusError(t, msgs[0].Value)
	if ev.Type != shops.StatusEventTypeError {
		t.Errorf("type: got %q want %q", ev.Type, shops.StatusEventTypeError)
	}
	if ev.Body.Error != shops.ErrorOk {
		t.Errorf("error: got %q want %q", ev.Body.Error, shops.ErrorOk)
	}
	if ev.CharacterId != testCharacterId {
		t.Errorf("characterId: got %d want %d", ev.CharacterId, testCharacterId)
	}
}

func TestBuyMesoPathEmitsOk(t *testing.T) {
	ctx := enterShop(t)
	p := confirmProcessor(t, ctx, nil,
		stubCommodities{cms: []commodities.Model{mesoCommodity(t, testPotionId, 50)}},
		confirmTestCharacter(t, 10000, useCompartment()))

	buf := message.NewBuffer()
	if err := p.Buy(buf)(testCharacterId)(0, testPotionId, 2, 0); err != nil {
		t.Fatalf("Buy returned an error: %v", err)
	}

	assertSingleOk(t, buf)
	if got := len(buf.GetAll()[characterMessage.EnvCommandTopic]); got != 1 {
		t.Errorf("expected one meso command, got %d", got)
	}
	if got := len(buf.GetAll()[compartmentMessage.EnvCommandTopic]); got != 1 {
		t.Errorf("expected one create command, got %d", got)
	}
}

func TestBuyRechargeablePathEmitsOk(t *testing.T) {
	ctx := enterShop(t)
	dataServer(t, map[uint32]consumableDoc{testStarId: {SlotMax: 200, UnitPrice: 5}}, nil)

	p := confirmProcessor(t, ctx, nil,
		stubCommodities{cms: []commodities.Model{mesoCommodity(t, testStarId, 1000)}},
		confirmTestCharacter(t, 10000, useCompartment()))

	buf := message.NewBuffer()
	if err := p.Buy(buf)(testCharacterId)(0, testStarId, 1, 0); err != nil {
		t.Fatalf("Buy returned an error: %v", err)
	}

	assertSingleOk(t, buf)
	if got := len(buf.GetAll()[compartmentMessage.EnvCommandTopic]); got != 1 {
		t.Errorf("expected one create command, got %d", got)
	}
}

// A refusal must not double up now that the success path also publishes.
func TestBuyRefusalEmitsExactlyOneEvent(t *testing.T) {
	ctx := enterShop(t)
	p := confirmProcessor(t, ctx, nil,
		stubCommodities{cms: []commodities.Model{mesoCommodity(t, testPotionId, 50)}},
		confirmTestCharacter(t, 10, useCompartment()))

	buf := message.NewBuffer()
	if err := p.Buy(buf)(testCharacterId)(0, testPotionId, 2, 0); err != nil {
		t.Fatalf("Buy returned an error: %v", err)
	}

	events := buf.GetAll()[shops.EnvStatusEventTopic]
	if len(events) != 1 {
		t.Fatalf("expected exactly one status event, got %d", len(events))
	}
	if ev := decodeStatusError(t, events[0].Value); ev.Body.Error != shops.ErrorNotEnoughMoney {
		t.Errorf("error: got %q want %q", ev.Body.Error, shops.ErrorNotEnoughMoney)
	}
	if got := len(buf.GetAll()[compartmentMessage.EnvCommandTopic]); got != 0 {
		t.Errorf("expected no compartment commands on refusal, got %d", got)
	}
}

func TestSellEmitsOk(t *testing.T) {
	ctx := enterShop(t)
	dataServer(t, nil, map[uint32]etcDoc{testEtcId: {Price: 100}})

	p := confirmProcessor(t, ctx, nil, nil,
		confirmTestCharacter(t, 0, etcCompartment(etcAsset(t, 5, testEtcId, 10))))

	buf := message.NewBuffer()
	if err := p.Sell(buf)(testCharacterId)(5, testEtcId, 3); err != nil {
		t.Fatalf("Sell returned an error: %v", err)
	}

	assertSingleOk(t, buf)
	if got := len(buf.GetAll()[characterMessage.EnvCommandTopic]); got != 1 {
		t.Errorf("expected one meso command, got %d", got)
	}
	if got := len(buf.GetAll()[compartmentMessage.EnvCommandTopic]); got != 1 {
		t.Errorf("expected one destroy command, got %d", got)
	}
}

func TestRechargeEmitsOk(t *testing.T) {
	ctx := enterShop(t)
	dataServer(t, map[uint32]consumableDoc{testStarId: {SlotMax: 200, UnitPrice: 5}}, nil)
	db := confirmTestDB(t)
	seedRechargerShop(t, db, ctx)

	p := confirmProcessor(t, ctx, db, nil,
		confirmTestCharacter(t, 100000, useCompartment(etcAsset(t, 1, testStarId, 10))))

	buf := message.NewBuffer()
	if err := p.Recharge(buf)(testCharacterId)(1); err != nil {
		t.Fatalf("Recharge returned an error: %v", err)
	}

	assertSingleOk(t, buf)
	if got := len(buf.GetAll()[characterMessage.EnvCommandTopic]); got != 1 {
		t.Errorf("expected one meso command, got %d", got)
	}
	if got := len(buf.GetAll()[compartmentMessage.EnvCommandTopic]); got != 1 {
		t.Errorf("expected one recharge command, got %d", got)
	}
}

// A full stack still consumed the client's outstanding request, so it is still
// owed a reply — this path previously returned with nothing published.
func TestRechargeAlreadyFullEmitsOk(t *testing.T) {
	ctx := enterShop(t)
	dataServer(t, map[uint32]consumableDoc{testStarId: {SlotMax: 200, UnitPrice: 5}}, nil)
	db := confirmTestDB(t)
	seedRechargerShop(t, db, ctx)

	p := confirmProcessor(t, ctx, db, nil,
		confirmTestCharacter(t, 100000, useCompartment(etcAsset(t, 1, testStarId, 200))))

	buf := message.NewBuffer()
	if err := p.Recharge(buf)(testCharacterId)(1); err != nil {
		t.Fatalf("Recharge returned an error: %v", err)
	}

	assertSingleOk(t, buf)
	if got := len(buf.GetAll()[compartmentMessage.EnvCommandTopic]); got != 0 {
		t.Errorf("expected no compartment commands for a full stack, got %d", got)
	}
}
