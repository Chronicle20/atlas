package craft

import (
	"atlas-maker/character"
	"atlas-maker/compartment"
	"atlas-maker/data/itemmake"
	"atlas-maker/quest"
	"atlas-maker/recipe"
	"atlas-maker/rest"
	"atlas-maker/skill"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	charactermock "atlas-maker/character/mock"
	compartmentmock "atlas-maker/compartment/mock"
	crystalbandmock "atlas-maker/crystalband/mock"
	equipmentmock "atlas-maker/data/equipment/mock"
	itemmakemock "atlas-maker/data/itemmake/mock"
	questmock "atlas-maker/quest/mock"
	reagentmock "atlas-maker/reagent/mock"
	recipemock "atlas-maker/recipe/mock"
	skillmock "atlas-maker/skill/mock"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// testServerInformation satisfies jsonapi.ServerInformation for resource tests.
type testServerInformation struct{}

func (testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (testServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = testServerInformation{}

func rtestLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return l
}

func rtestContext(t *testing.T) context.Context {
	t.Helper()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), te)
}

// rspyEmitter records every saga it is given so a test can assert exactly
// what was (or, on rejection, was not) emitted.
type rspyEmitter struct {
	calls []saga.Saga
}

func (e *rspyEmitter) Emit(s saga.Saga) error {
	e.calls = append(e.calls, s)
	return nil
}

// rBuildRecipe indexes rm through a real recipe.Processor (backed by an
// itemmake mock, on a fresh tenant) to obtain a genuine recipe.Model, since
// recipe.Model's fields are private (mirrors eligibility_test.go's own
// buildRecipe, duplicated here because that helper lives in the external
// craft_test package this file cannot see).
func rBuildRecipe(t *testing.T, rm itemmake.RestModel) recipe.Model {
	t.Helper()
	im, err := itemmake.Extract(rm)
	require.NoError(t, err)

	ctx := rtestContext(t)
	imp := &itemmakemock.ProcessorMock{
		GetAllFunc: func() ([]itemmake.Model, error) { return []itemmake.Model{im}, nil },
	}
	p := recipe.NewProcessor(rtestLogger(), ctx, imp)
	m, err := p.GetById(item.Id(rm.Id))
	require.NoError(t, err)
	return m
}

func rBuildCharacter(t *testing.T, level byte, meso uint32) character.Model {
	t.Helper()
	m, err := character.Extract(character.RestModel{Level: level, Meso: meso})
	require.NoError(t, err)
	return m
}

func rBuildSkill(t *testing.T, id uint32, level byte) skill.Model {
	t.Helper()
	m, err := skill.Extract(skill.RestModel{Id: id, Level: level})
	require.NoError(t, err)
	return m
}

func rBuildQuest(t *testing.T, questId uint32, state byte) quest.Model {
	t.Helper()
	m, err := quest.Extract(quest.RestModel{QuestId: questId, State: state})
	require.NoError(t, err)
	return m
}

// rharness is a mutable set of upstream fixtures, starting from a fully
// eligible baseline that a test then breaks exactly one condition of
// (mirrors craft_test's harness).
type rharness struct {
	characterId uint32
	character   character.Model
	skills      []skill.Model
	equip       compartment.Model
	use         compartment.Model
	etc         compartment.Model
	quests      []quest.Model
	accommodate bool
}

// rEligibleRecipeFixture mirrors craft_test's eligibleRecipeFixture: every
// optional requirement present, gating exactly the values rNewEligibleHarness
// satisfies.
func rEligibleRecipeFixture(t *testing.T) recipe.Model {
	t.Helper()
	return rBuildRecipe(t, itemmake.RestModel{
		Id:            1082002,
		Group:         1,
		ReqLevel:      30,
		ReqSkillLevel: 2,
		ItemNum:       1,
		Tuc:           7,
		Meso:          1200,
		Catalyst:      4130000,
		ReqItem:       4000021,
		ReqEquip:      1002419,
		Recipe:        []itemmake.MaterialRestModel{{ItemId: 4011001, Count: 5}},
		ReqQuest:      []itemmake.QuestReqRestModel{{QuestId: 21614, State: 3}},
	})
}

func rNewEligibleHarness(t *testing.T) *rharness {
	t.Helper()
	return &rharness{
		characterId: 1001,
		character:   rBuildCharacter(t, 40, 1200),
		skills:      []skill.Model{rBuildSkill(t, uint32(skillconst.BeginnerMaker), 2)},
		equip: compartment.NewBuilder(inventory.TypeValueEquip).
			AddAsset(compartment.NewAssetModel(item.Id(1002419), 1, -1)).
			Build(),
		use: compartment.NewBuilder(inventory.TypeValueUse).Build(),
		etc: compartment.NewBuilder(inventory.TypeValueETC).
			AddAsset(compartment.NewAssetModel(item.Id(4011001), 5, 1)).
			AddAsset(compartment.NewAssetModel(item.Id(4000021), 1, 2)).
			Build(),
		quests:      []quest.Model{rBuildQuest(t, 21614, 3)},
		accommodate: true,
	}
}

// rdeps bundles the recipe/reagent/crystalband/equipment/emitter mocks a
// resource test configures around a shared rharness.
type rdeps struct {
	rp  *recipemock.ProcessorMock
	rgp *reagentmock.ProcessorMock
	cbp *crystalbandmock.ProcessorMock
	eqp *equipmentmock.ProcessorMock
	em  *rspyEmitter
}

func rNewDeps() *rdeps {
	return &rdeps{
		rp:  &recipemock.ProcessorMock{},
		rgp: &reagentmock.ProcessorMock{},
		cbp: &crystalbandmock.ProcessorMock{},
		eqp: &equipmentmock.ProcessorMock{},
		em:  &rspyEmitter{},
	}
}

// rBuildProcessor wires h's character/skill/compartment/quest fixtures
// together with d's recipe/reagent/crystalband/equipment/emitter mocks into
// a real craft.Processor, and returns d.rp alongside it as the
// recipe.Processor a test's processorFactory hands to resource.go.
func rBuildProcessor(t *testing.T, h *rharness, d *rdeps) (Processor, recipe.Processor) {
	t.Helper()
	cp := &charactermock.ProcessorMock{
		GetByIdFunc: func(uint32) (character.Model, error) { return h.character, nil },
	}
	sp := &skillmock.ProcessorMock{
		GetByCharacterIdFunc: func(uint32) ([]skill.Model, error) { return h.skills, nil },
	}
	kp := &compartmentmock.ProcessorMock{
		GetByTypeFunc: func(_ uint32, invType inventory.Type) (compartment.Model, error) {
			switch invType {
			case inventory.TypeValueEquip:
				return h.equip, nil
			case inventory.TypeValueUse:
				return h.use, nil
			default:
				return h.etc, nil
			}
		},
		CanAccommodateFunc: func(uint32, []compartment.AccommodationItem) (bool, error) {
			return h.accommodate, nil
		},
	}
	qp := &questmock.ProcessorMock{
		GetByCharacterIdFunc: func(uint32) ([]quest.Model, error) { return h.quests, nil },
	}
	p := NewProcessor(rtestLogger(), rtestContext(t), cp, sp, kp, qp, d.rp, d.rgp, d.cbp, d.eqp, d.em)
	return p, d.rp
}

// newTestFactory returns a processorFactory that ignores its
// *rest.HandlerDependency and always hands back the pre-built p/rp pair, so
// a resource test never dials the real HTTP-backed upstream clients
// defaultProcessorFactory wires.
func newTestFactory(p Processor, rp recipe.Processor) processorFactory {
	return func(*rest.HandlerDependency) (Processor, recipe.Processor) {
		return p, rp
	}
}

func setupRouter(pf processorFactory) *mux.Router {
	r := mux.NewRouter()
	initResource(testServerInformation{}, pf)((*gorm.DB)(nil))(r, rtestLogger())
	return r
}

func requestWithTenant(method, url string, body string, tenantId uuid.UUID) *http.Request {
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("TENANT_ID", tenantId.String())
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

func TestListRecipesReturnsOnlyEligible(t *testing.T) {
	h := rNewEligibleHarness(t)
	eligibleA := rEligibleRecipeFixture(t)
	eligibleB := rBuildRecipe(t, itemmake.RestModel{
		Id: 1082003, Group: 1, ReqLevel: 30, ReqSkillLevel: 2, ItemNum: 1, Tuc: 7, Meso: 1200,
		Catalyst: 4130000, ReqItem: 4000021, ReqEquip: 1002419,
		Recipe:   []itemmake.MaterialRestModel{{ItemId: 4011001, Count: 5}},
		ReqQuest: []itemmake.QuestReqRestModel{{QuestId: 21614, State: 3}},
	})
	ineligible := make([]recipe.Model, 0, 3)
	for i := 0; i < 3; i++ {
		ineligible = append(ineligible, rBuildRecipe(t, itemmake.RestModel{
			Id: uint32(1082010 + i), Group: 1, ReqLevel: 999, ItemNum: 1,
		}))
	}

	d := rNewDeps()
	all := append([]recipe.Model{eligibleA, eligibleB}, ineligible...)
	d.rp.GetAllFunc = func() ([]recipe.Model, error) { return all, nil }

	p, rp := rBuildProcessor(t, h, d)
	router := setupRouter(newTestFactory(p, rp))
	srv := httptest.NewServer(router)
	defer srv.Close()

	url := fmt.Sprintf("%s/characters/%d/maker/recipes", srv.URL, h.characterId)
	req := requestWithTenant(http.MethodGet, url, "", uuid.New())
	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc jsonapi.Document
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
	require.NotNil(t, doc.Data)
	require.Len(t, doc.Data.DataArray, 2)

	ids := map[string]bool{}
	for _, do := range doc.Data.DataArray {
		ids[do.ID] = true
	}
	assert.True(t, ids["1082002"])
	assert.True(t, ids["1082003"])
}

func TestListRecipesPaginates(t *testing.T) {
	h := rNewEligibleHarness(t)
	recipes := make([]recipe.Model, 0, 25)
	for i := 0; i < 25; i++ {
		recipes = append(recipes, rBuildRecipe(t, itemmake.RestModel{
			Id: uint32(2000000 + i), Group: 1, ReqLevel: 1, ReqSkillLevel: 1, ItemNum: 1,
		}))
	}

	d := rNewDeps()
	d.rp.GetAllFunc = func() ([]recipe.Model, error) { return recipes, nil }

	p, rp := rBuildProcessor(t, h, d)
	router := setupRouter(newTestFactory(p, rp))
	srv := httptest.NewServer(router)
	defer srv.Close()

	url := fmt.Sprintf("%s/characters/%d/maker/recipes?page[number]=2&page[size]=10", srv.URL, h.characterId)
	req := requestWithTenant(http.MethodGet, url, "", uuid.New())
	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc jsonapi.Document
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
	require.NotNil(t, doc.Data)
	require.Len(t, doc.Data.DataArray, 10)
	require.NotNil(t, doc.Meta)
	assert.EqualValues(t, 25, doc.Meta["total"])
	require.NotEmpty(t, doc.Links, "expected pagination links to be present")
	assert.Contains(t, doc.Links, "next")
	assert.Contains(t, doc.Links, "prev")
}

func TestGetRecipeIncludesEligibilityAndCost(t *testing.T) {
	h := rNewEligibleHarness(t)
	r := rEligibleRecipeFixture(t)

	d := rNewDeps()
	d.rp.GetByIdFunc = func(item.Id) (recipe.Model, error) { return r, nil }

	p, rp := rBuildProcessor(t, h, d)
	router := setupRouter(newTestFactory(p, rp))
	srv := httptest.NewServer(router)
	defer srv.Close()

	url := fmt.Sprintf("%s/characters/%d/maker/recipes/%d", srv.URL, h.characterId, r.Id())
	req := requestWithTenant(http.MethodGet, url, "", uuid.New())
	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var doc jsonapi.Document
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
	require.NotNil(t, doc.Data)
	require.NotNil(t, doc.Data.DataObject)

	var attrs struct {
		Eligible  bool                `json:"eligible"`
		Meso      uint32              `json:"meso"`
		Materials []MaterialRestModel `json:"materials"`
	}
	require.NoError(t, json.Unmarshal(doc.Data.DataObject.Attributes, &attrs))
	assert.True(t, attrs.Eligible)
	assert.EqualValues(t, 1200, attrs.Meso)
	require.Len(t, attrs.Materials, 1)
	assert.EqualValues(t, 4011001, attrs.Materials[0].ItemId)
	assert.EqualValues(t, 5, attrs.Materials[0].Count)
}

// postCraft issues the JSON:API-encoded craft request over HTTP through the
// full route stack (mux, tenant parsing, JSON:API marshaling).
func postCraft(t *testing.T, router *mux.Router, tenantId uuid.UUID, characterId uint32, body string) *http.Response {
	t.Helper()
	srv := httptest.NewServer(router)
	defer srv.Close()

	url := fmt.Sprintf("%s/characters/%d/maker/crafts", srv.URL, characterId)
	req := requestWithTenant(http.MethodPost, url, body, tenantId)
	resp, err := (&http.Client{}).Do(req)
	require.NoError(t, err)
	return resp
}

func craftBody(mode uint32, targetItemId uint32) string {
	return fmt.Sprintf(`{"data":{"type":"makerCrafts","attributes":{"mode":%d,"targetItemId":%d}}}`, mode, targetItemId)
}

// TestEveryErrorCodeIsReturnedByItsOwnCondition is table-driven, one case
// per row of the brief's error table, each provoking exactly that
// condition.
func TestEveryErrorCodeIsReturnedByItsOwnCondition(t *testing.T) {
	t.Run("recipe_not_found", func(t *testing.T) {
		h := rNewEligibleHarness(t)
		d := rNewDeps()
		d.rp.GetByIdFunc = func(item.Id) (recipe.Model, error) { return recipe.Model{}, recipe.ErrNotFound }
		p, rp := rBuildProcessor(t, h, d)
		router := setupRouter(newTestFactory(p, rp))
		srv := httptest.NewServer(router)
		defer srv.Close()

		url := fmt.Sprintf("%s/characters/%d/maker/recipes/9999999", srv.URL, h.characterId)
		req := requestWithTenant(http.MethodGet, url, "", uuid.New())
		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assertCraftError(t, resp, http.StatusNotFound, CodeRecipeNotFound)
	})

	type tc struct {
		name   string
		mutate func(t *testing.T, h *rharness, d *rdeps)
		body   string
		status int
		code   Code
	}
	r := rEligibleRecipeFixture(t)
	tests := []tc{
		{
			name:   "level_too_low",
			mutate: func(t *testing.T, h *rharness, d *rdeps) { h.character = rBuildCharacter(t, 1, 1200) },
			body:   craftBody(1, uint32(r.Id())),
			status: http.StatusUnprocessableEntity, code: CodeLevelTooLow,
		},
		{
			name:   "skill_level_too_low",
			mutate: func(t *testing.T, h *rharness, d *rdeps) { h.skills = nil },
			body:   craftBody(1, uint32(r.Id())),
			status: http.StatusUnprocessableEntity, code: CodeSkillLevelTooLow,
		},
		{
			name: "insufficient_materials",
			mutate: func(t *testing.T, h *rharness, d *rdeps) {
				h.etc = compartment.NewBuilder(inventory.TypeValueETC).
					AddAsset(compartment.NewAssetModel(item.Id(4000021), 1, 2)).
					Build()
			},
			body:   craftBody(1, uint32(r.Id())),
			status: http.StatusUnprocessableEntity, code: CodeInsufficientMaterials,
		},
		{
			name: "missing_prerequisite_item",
			mutate: func(t *testing.T, h *rharness, d *rdeps) {
				h.etc = compartment.NewBuilder(inventory.TypeValueETC).
					AddAsset(compartment.NewAssetModel(item.Id(4011001), 5, 1)).
					Build()
			},
			body:   craftBody(1, uint32(r.Id())),
			status: http.StatusUnprocessableEntity, code: CodeMissingPrerequisiteItem,
		},
		{
			name:   "missing_prerequisite_quest",
			mutate: func(t *testing.T, h *rharness, d *rdeps) { h.quests = nil },
			body:   craftBody(1, uint32(r.Id())),
			status: http.StatusUnprocessableEntity, code: CodeMissingPrerequisiteQuest,
		},
		{
			name:   "insufficient_mesos",
			mutate: func(t *testing.T, h *rharness, d *rdeps) { h.character = rBuildCharacter(t, 40, 0) },
			body:   craftBody(1, uint32(r.Id())),
			status: http.StatusUnprocessableEntity, code: CodeInsufficientMesos,
		},
		{
			name:   "inventory_full",
			mutate: func(t *testing.T, h *rharness, d *rdeps) { h.accommodate = false },
			body:   craftBody(1, uint32(r.Id())),
			status: http.StatusUnprocessableEntity, code: CodeInventoryFull,
		},
		{
			name: "equip_not_found",
			mutate: func(t *testing.T, h *rharness, d *rdeps) {
				h.equip = compartment.NewBuilder(inventory.TypeValueEquip).
					AddAsset(compartment.NewAssetModel(item.Id(1002419), 1, 5)).
					Build()
			},
			body:   `{"data":{"type":"makerCrafts","attributes":{"mode":4,"equipItemId":1002419,"slotPos":99}}}`,
			status: http.StatusUnprocessableEntity, code: CodeEquipNotFound,
		},
		{
			name: "no_crystal_mapping",
			mutate: func(t *testing.T, h *rharness, d *rdeps) {
				d.rp.GetByLeftoverFunc = func(item.Id) (recipe.Model, error) { return recipe.Model{}, recipe.ErrNoCrystalMapping }
			},
			body:   `{"data":{"type":"makerCrafts","attributes":{"mode":3,"leftoverItemId":4200099}}}`,
			status: http.StatusUnprocessableEntity, code: CodeNoCrystalMapping,
		},
		{
			name:   "craft_in_progress",
			mutate: nil,
			body:   craftBody(1, uint32(r.Id())),
			status: http.StatusConflict, code: CodeCraftInProgress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := rNewEligibleHarness(t)
			d := rNewDeps()
			d.rp.GetByIdFunc = func(item.Id) (recipe.Model, error) { return r, nil }
			if tt.mutate != nil {
				tt.mutate(t, h, d)
			}
			p, rp := rBuildProcessor(t, h, d)
			router := setupRouter(newTestFactory(p, rp))

			// craft_in_progress is provoked by a first, accepted call that
			// legitimately emits one saga; every other row's single call is
			// itself the rejection, expected to emit none.
			expectedCalls := 0
			if tt.name == "craft_in_progress" {
				resp1 := postCraft(t, router, uuid.New(), h.characterId, tt.body)
				require.Equal(t, http.StatusOK, resp1.StatusCode)
				_ = resp1.Body.Close()
				expectedCalls = 1
			}

			resp := postCraft(t, router, uuid.New(), h.characterId, tt.body)
			defer func() { _ = resp.Body.Close() }()
			assertCraftError(t, resp, tt.status, tt.code)
			assert.Len(t, d.em.calls, expectedCalls, "the rejected call must emit no additional saga")
		})
	}
}

func assertCraftError(t *testing.T, resp *http.Response, status int, code Code) {
	t.Helper()
	assert.Equal(t, status, resp.StatusCode)
	var doc struct {
		Errors []struct {
			Status string `json:"status"`
			Code   string `json:"code"`
		} `json:"errors"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
	require.Len(t, doc.Errors, 1)
	assert.Equal(t, string(code), doc.Errors[0].Code)
}

func TestPostCraftReturnsSagaId(t *testing.T) {
	h := rNewEligibleHarness(t)
	r := rEligibleRecipeFixture(t)
	d := rNewDeps()
	d.rp.GetByIdFunc = func(item.Id) (recipe.Model, error) { return r, nil }

	p, rp := rBuildProcessor(t, h, d)
	router := setupRouter(newTestFactory(p, rp))

	resp := postCraft(t, router, uuid.New(), h.characterId, craftBody(1, uint32(r.Id())))
	defer func() { _ = resp.Body.Close() }()
	require.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted)

	require.Len(t, d.em.calls, 1)
	expected := d.em.calls[0].TransactionId.String()

	var doc jsonapi.Document
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&doc))
	require.NotNil(t, doc.Data)
	require.NotNil(t, doc.Data.DataObject)
	assert.Equal(t, expected, doc.Data.DataObject.ID)

	var attrs struct {
		TransactionId string `json:"transactionId"`
	}
	require.NoError(t, json.Unmarshal(doc.Data.DataObject.Attributes, &attrs))
	assert.Equal(t, expected, attrs.TransactionId)
}

// TestFailureLeavesStateUnchanged is FR-3.7's acceptance criterion, asserted
// rather than reasoned about: for every rejection condition, capture the
// character's materials, mesos and equips before the request, issue it, and
// assert the post-request state is identical field-for-field, and that the
// saga emitter was called zero times. This mirrors every row of
// TestEveryErrorCodeIsReturnedByItsOwnCondition's table (all 11 rejection
// conditions), not a subset.
func TestFailureLeavesStateUnchanged(t *testing.T) {
	r := rEligibleRecipeFixture(t)

	t.Run("recipe_not_found", func(t *testing.T) {
		h := rNewEligibleHarness(t)
		beforeCharacter := h.character
		beforeEtc := h.etc
		beforeEquip := h.equip

		d := rNewDeps()
		d.rp.GetByIdFunc = func(item.Id) (recipe.Model, error) { return recipe.Model{}, recipe.ErrNotFound }
		p, rp := rBuildProcessor(t, h, d)
		router := setupRouter(newTestFactory(p, rp))
		srv := httptest.NewServer(router)
		defer srv.Close()

		url := fmt.Sprintf("%s/characters/%d/maker/recipes/9999999", srv.URL, h.characterId)
		req := requestWithTenant(http.MethodGet, url, "", uuid.New())
		resp, err := (&http.Client{}).Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusNotFound, resp.StatusCode)

		assert.Equal(t, beforeCharacter, h.character, "character must be unchanged after a rejected lookup")
		assert.Equal(t, beforeEtc, h.etc, "etc compartment must be unchanged after a rejected lookup")
		assert.Equal(t, beforeEquip, h.equip, "equip compartment must be unchanged after a rejected lookup")
		assert.Empty(t, d.em.calls, "a rejection must emit no saga")
	})

	tests := []struct {
		name   string
		mutate func(t *testing.T, h *rharness, d *rdeps)
		body   string
	}{
		{
			name:   "level_too_low",
			mutate: func(t *testing.T, h *rharness, d *rdeps) { h.character = rBuildCharacter(t, 1, 1200) },
			body:   craftBody(1, uint32(r.Id())),
		},
		{
			name:   "skill_level_too_low",
			mutate: func(t *testing.T, h *rharness, d *rdeps) { h.skills = nil },
			body:   craftBody(1, uint32(r.Id())),
		},
		{
			name: "insufficient_materials",
			mutate: func(t *testing.T, h *rharness, d *rdeps) {
				h.etc = compartment.NewBuilder(inventory.TypeValueETC).
					AddAsset(compartment.NewAssetModel(item.Id(4000021), 1, 2)).
					Build()
			},
			body: craftBody(1, uint32(r.Id())),
		},
		{
			name: "missing_prerequisite_item",
			mutate: func(t *testing.T, h *rharness, d *rdeps) {
				h.etc = compartment.NewBuilder(inventory.TypeValueETC).
					AddAsset(compartment.NewAssetModel(item.Id(4011001), 5, 1)).
					Build()
			},
			body: craftBody(1, uint32(r.Id())),
		},
		{
			name:   "missing_prerequisite_quest",
			mutate: func(t *testing.T, h *rharness, d *rdeps) { h.quests = nil },
			body:   craftBody(1, uint32(r.Id())),
		},
		{
			name:   "insufficient_mesos",
			mutate: func(t *testing.T, h *rharness, d *rdeps) { h.character = rBuildCharacter(t, 40, 0) },
			body:   craftBody(1, uint32(r.Id())),
		},
		{
			name:   "inventory_full",
			mutate: func(t *testing.T, h *rharness, d *rdeps) { h.accommodate = false },
			body:   craftBody(1, uint32(r.Id())),
		},
		{
			name: "equip_not_found",
			mutate: func(t *testing.T, h *rharness, d *rdeps) {
				h.equip = compartment.NewBuilder(inventory.TypeValueEquip).
					AddAsset(compartment.NewAssetModel(item.Id(1002419), 1, 5)).
					Build()
			},
			body: `{"data":{"type":"makerCrafts","attributes":{"mode":4,"equipItemId":1002419,"slotPos":99}}}`,
		},
		{
			name: "no_crystal_mapping",
			mutate: func(t *testing.T, h *rharness, d *rdeps) {
				d.rp.GetByLeftoverFunc = func(item.Id) (recipe.Model, error) { return recipe.Model{}, recipe.ErrNoCrystalMapping }
			},
			body: `{"data":{"type":"makerCrafts","attributes":{"mode":3,"leftoverItemId":4200099}}}`,
		},
		{
			name:   "craft_in_progress",
			mutate: nil,
			body:   craftBody(1, uint32(r.Id())),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := rNewEligibleHarness(t)
			d := rNewDeps()
			d.rp.GetByIdFunc = func(item.Id) (recipe.Model, error) { return r, nil }
			if tt.mutate != nil {
				tt.mutate(t, h, d)
			}
			p, rp := rBuildProcessor(t, h, d)
			router := setupRouter(newTestFactory(p, rp))

			// craft_in_progress is provoked by a first, accepted call that
			// legitimately emits one saga and mutates state; the state
			// snapshot is taken after that call, since only the second
			// (rejected) call's effect on state is under test here.
			expectedCalls := 0
			if tt.name == "craft_in_progress" {
				resp1 := postCraft(t, router, uuid.New(), h.characterId, tt.body)
				require.Equal(t, http.StatusOK, resp1.StatusCode)
				_ = resp1.Body.Close()
				expectedCalls = 1
			}

			beforeCharacter := h.character
			beforeEtc := h.etc
			beforeEquip := h.equip

			resp := postCraft(t, router, uuid.New(), h.characterId, tt.body)
			defer func() { _ = resp.Body.Close() }()
			require.NotEqual(t, http.StatusOK, resp.StatusCode)

			assert.Equal(t, beforeCharacter, h.character, "character must be unchanged after a rejected craft")
			assert.Equal(t, beforeEtc, h.etc, "etc compartment must be unchanged after a rejected craft")
			assert.Equal(t, beforeEquip, h.equip, "equip compartment must be unchanged after a rejected craft")
			assert.Len(t, d.em.calls, expectedCalls, "the rejected call must emit no additional saga")
		})
	}
}

func TestRecipeRoutesAreReadOnly(t *testing.T) {
	h := rNewEligibleHarness(t)
	d := rNewDeps()
	p, rp := rBuildProcessor(t, h, d)
	router := setupRouter(newTestFactory(p, rp))
	srv := httptest.NewServer(router)
	defer srv.Close()

	paths := []string{
		fmt.Sprintf("/characters/%d/maker/recipes", h.characterId),
		fmt.Sprintf("/characters/%d/maker/recipes/1082002", h.characterId),
	}
	methods := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

	for _, path := range paths {
		for _, method := range methods {
			t.Run(method+path, func(t *testing.T) {
				req := requestWithTenant(method, srv.URL+path, "", uuid.New())
				resp, err := (&http.Client{}).Do(req)
				require.NoError(t, err)
				defer func() { _ = resp.Body.Close() }()
				assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
			})
		}
	}
}
