// Package coupon_test drives the admin REST surface through the REAL routers
// (coupon.InitResource, batch.InitResource, redemption.InitResource) over real
// HTTP requests.
//
// It is an EXTERNAL test package on purpose: package coupon/batch imports
// package coupon, so a test inside package coupon could not register the batch
// routes. Everything below therefore uses the exported API only — the same
// surface the atlas-ui client sees.
//
// NOTE ON THE HARNESS AND WHAT IT DOES / DOES NOT PROVE.
//
// These tests run against gorm's SQLite in-memory driver via
// databasetest.NewInMemoryTenantDB, NOT Postgres, per the human ruling on this
// branch. databasetest caps the pool at ONE connection, so nothing here can
// run two requests against the database simultaneously. In particular
// TestCreateBatchGenerates500UniqueCodes is a THROUGHPUT AND UNIQUENESS test,
// not a concurrency test: it shows that one 500-code generation produces 500
// distinct codes and a batch that claims 500, and it is not evidence about two
// simultaneous generations.
package coupon_test

import (
	"atlas-cashshop/cashshop/commodity"
	"atlas-cashshop/coupon"
	"atlas-cashshop/coupon/batch"
	"atlas-cashshop/coupon/redemption"
	"atlas-cashshop/coupon/reward"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type testServerInformation struct{}

func (t *testServerInformation) GetBaseURL() string { return "http://localhost:8080" }
func (t *testServerInformation) GetPrefix() string  { return "/api/" }

var _ jsonapi.ServerInformation = &testServerInformation{}

func ptrU32(v uint32) *uint32        { return &v }
func ptrTime(t time.Time) *time.Time { return &t }

// newEnv migrates the three coupon tables, registers all three resources on one
// router, and returns a live server plus the tenant every fixture belongs to.
func newEnv(t *testing.T) (*httptest.Server, *gorm.DB, tenant.Model) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, coupon.Migration, batch.Migration, redemption.Migration)

	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.ErrorLevel)

	r := mux.NewRouter()
	si := &testServerInformation{}
	coupon.InitResource(si)(db)(r, l)
	batch.InitResource(si)(db)(r, l)
	redemption.InitResource(si)(db)(r, l)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return srv, db, tm
}

func do(t *testing.T, srv *httptest.Server, tm tenant.Model, method, path string, body []byte) (*http.Response, []byte) {
	t.Helper()
	var rdr *bytes.Reader
	if body == nil {
		rdr = bytes.NewReader(nil)
	} else {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, srv.URL+path, rdr)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("TENANT_ID", tm.Id().String())
	req.Header.Set("REGION", tm.Region())
	req.Header.Set("MAJOR_VERSION", fmt.Sprintf("%d", tm.MajorVersion()))
	req.Header.Set("MINOR_VERSION", fmt.Sprintf("%d", tm.MinorVersion()))

	resp, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	require.NoError(t, err)
	return resp, buf.Bytes()
}

func marshalCoupon(t *testing.T, rm coupon.RestModel) []byte {
	t.Helper()
	b, err := jsonapi.Marshal(rm)
	require.NoError(t, err)
	return b
}

// patchBody writes a PATCH body from a RAW attributes document, because the
// whole point of these tests is which keys are PRESENT: a Go struct with
// omitempty cannot express "expiresAt is absent" versus "expiresAt is null",
// and that distinction is the semantics under test.
func patchBody(attributes string) []byte {
	return []byte(`{"data":{"type":"coupons","attributes":` + attributes + `}}`)
}

func marshalBatch(t *testing.T, rm batch.RestModel) []byte {
	t.Helper()
	b, err := jsonapi.Marshal(rm)
	require.NoError(t, err)
	return b
}

func decodeDoc(t *testing.T, body []byte) jsonapi.Document {
	t.Helper()
	var doc jsonapi.Document
	require.NoError(t, json.Unmarshal(body, &doc))
	return doc
}

func attr(t *testing.T, obj jsonapi.Data, name string) interface{} {
	t.Helper()
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(obj.Attributes, &m))
	return m[name]
}

func currencyBundle() reward.Rewards {
	return reward.Rewards{reward.NewCurrencyReward(1, 100)}
}

// seedCoupon inserts a coupon directly, bypassing REST, so a test can arrange
// state a route deliberately refuses to create.
func seedCoupon(t *testing.T, db *gorm.DB, tm tenant.Model, b *coupon.Builder) coupon.Model {
	t.Helper()
	m, err := b.Build()
	require.NoError(t, err)
	created, err := coupon.CreateEntity(db, tm, m)
	require.NoError(t, err)
	return created
}

func seedRedemption(t *testing.T, db *gorm.DB, tm tenant.Model, couponId uuid.UUID, accountId uint32) redemption.Model {
	t.Helper()
	rm, err := redemption.NewBuilder(couponId, accountId, 4242).
		SetTransactionId(uuid.New()).
		SetRewardsGranted(currencyBundle()).
		SetRedeemedAt(time.Now()).
		Build()
	require.NoError(t, err)
	created, err := redemption.Create(db, tm, rm)
	require.NoError(t, err)
	return created
}

// --- POST /coupons -----------------------------------------------------------

func TestCreateCouponWithAnExplicitCode(t *testing.T) {
	srv, _, tm := newEnv(t)

	body := marshalCoupon(t, coupon.RestModel{Code: "maple2026", Description: "launch", Active: true, Rewards: currencyBundle()})
	resp, out := do(t, srv, tm, http.MethodPost, "/coupons", body)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(out))

	doc := decodeDoc(t, out)
	require.NotNil(t, doc.Data)
	require.NotNil(t, doc.Data.DataObject)
	assert.Equal(t, "coupons", doc.Data.DataObject.Type)
	assert.Equal(t, "MAPLE2026", attr(t, *doc.Data.DataObject, "code"),
		"the stored code must be the normalized form")
}

func TestCreateCouponRejectsADuplicateCodeWith409(t *testing.T) {
	srv, _, tm := newEnv(t)

	first := marshalCoupon(t, coupon.RestModel{Code: "DUPE", Active: true, Rewards: currencyBundle()})
	resp, out := do(t, srv, tm, http.MethodPost, "/coupons", first)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(out))

	for name, code := range map[string]string{
		"exact":              "DUPE",
		"lowercase variant":  "dupe",
		"whitespace variant": "  DuPe  ",
	} {
		t.Run(name, func(t *testing.T) {
			again := marshalCoupon(t, coupon.RestModel{Code: code, Active: true, Rewards: currencyBundle()})
			resp, out := do(t, srv, tm, http.MethodPost, "/coupons", again)
			assert.Equal(t, http.StatusConflict, resp.StatusCode, string(out))
		})
	}
}

func TestCreateCouponWithABlankCodeGeneratesOne(t *testing.T) {
	srv, _, tm := newEnv(t)

	body := marshalCoupon(t, coupon.RestModel{Code: "   ", Active: true, Rewards: currencyBundle()})
	resp, out := do(t, srv, tm, http.MethodPost, "/coupons", body)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(out))

	code, ok := attr(t, *decodeDoc(t, out).Data.DataObject, "code").(string)
	require.True(t, ok)
	assert.Len(t, code, coupon.DefaultGeneratedCodeLength)
	assert.NotContains(t, strings.ToUpper(code), "O")
}

func TestCreateCouponRejectsExpiresAtAtOrBeforeStartsAtWith422(t *testing.T) {
	srv, _, tm := newEnv(t)
	now := time.Now()

	for name, expires := range map[string]time.Time{
		"equal":  now,
		"before": now.Add(-time.Hour),
	} {
		t.Run(name, func(t *testing.T) {
			body := marshalCoupon(t, coupon.RestModel{
				Code: "WINDOW" + name, Active: true, Rewards: currencyBundle(),
				StartsAt: ptrTime(now), ExpiresAt: ptrTime(expires),
			})
			resp, out := do(t, srv, tm, http.MethodPost, "/coupons", body)
			assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode, string(out))
		})
	}
}

func TestCreateCouponRejectsAnEmptyRewardBundleWith422(t *testing.T) {
	srv, _, tm := newEnv(t)
	body := marshalCoupon(t, coupon.RestModel{Code: "NOREWARD", Active: true})
	resp, out := do(t, srv, tm, http.MethodPost, "/coupons", body)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode, string(out))
}

// --- unknown commodity serial ------------------------------------------------

// commodityStub stands in for atlas-data. Serials in known are served; every
// other serial 404s, which is exactly what requests.ErrNotFound is built from.
func commodityStub(t *testing.T, known map[uint32]uint32) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var serial uint32
		if _, err := fmt.Sscanf(r.URL.Path, "/api/data/commodity/items/%d", &serial); err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		itemId, ok := known[serial]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body, err := jsonapi.Marshal(commodity.RestModel{Id: serial, ItemId: itemId, Count: 1, OnSale: true})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/api/")
}

func TestCreateCouponRejectsAnUnknownCommoditySerialWith422(t *testing.T) {
	// 40000 is served; 999999 is not.
	commodityStub(t, map[uint32]uint32{40000: 5000000})
	srv, _, tm := newEnv(t)

	t.Run("known serial is accepted", func(t *testing.T) {
		body := marshalCoupon(t, coupon.RestModel{
			Code: "GOODSERIAL", Active: true,
			Rewards: reward.Rewards{reward.NewCashItemReward(40000, 1)},
		})
		resp, out := do(t, srv, tm, http.MethodPost, "/coupons", body)
		assert.Equal(t, http.StatusOK, resp.StatusCode, string(out))
	})

	t.Run("unknown serial is 422", func(t *testing.T) {
		body := marshalCoupon(t, coupon.RestModel{
			Code: "BADSERIAL", Active: true,
			Rewards: reward.Rewards{reward.NewCashItemReward(999999, 1)},
		})
		resp, out := do(t, srv, tm, http.MethodPost, "/coupons", body)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode, string(out))
	})
}

// --- PATCH /coupons/{id} -----------------------------------------------------

func TestUpdateCouponRejectsMaxUsesBelowTheRedemptionCountWith422(t *testing.T) {
	srv, db, tm := newEnv(t)
	c := seedCoupon(t, db, tm, coupon.NewBuilder("USED").SetMaxUses(ptrU32(5)).SetRewards(currencyBundle()))

	// redemption_count is owned by reserveUse; set it directly to arrange the
	// state a PATCH has to refuse.
	require.NoError(t, db.Model(&coupon.Entity{}).Where("id = ?", c.Id()).UpdateColumn("redemption_count", 3).Error)

	resp, out := do(t, srv, tm, http.MethodPatch, "/coupons/"+c.Id().String(), patchBody(`{"maxUses":2}`))
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode, string(out))

	t.Run("at or above the count is accepted", func(t *testing.T) {
		resp, out := do(t, srv, tm, http.MethodPatch, "/coupons/"+c.Id().String(), patchBody(`{"maxUses":3}`))
		require.Equal(t, http.StatusOK, resp.StatusCode, string(out))
		assert.EqualValues(t, 3, attr(t, *decodeDoc(t, out).Data.DataObject, "maxUses"))
	})
}

func TestUpdateCouponEditsTheEditableFields(t *testing.T) {
	srv, db, tm := newEnv(t)
	c := seedCoupon(t, db, tm, coupon.NewBuilder("EDITME").SetDescription("before").SetRewards(currencyBundle()))

	body := patchBody(`{"code":"IGNORED","description":"after","active":false}`)
	resp, out := do(t, srv, tm, http.MethodPatch, "/coupons/"+c.Id().String(), body)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(out))

	obj := *decodeDoc(t, out).Data.DataObject
	assert.Equal(t, "after", attr(t, obj, "description"))
	assert.Equal(t, false, attr(t, obj, "active"))
	assert.Equal(t, "EDITME", attr(t, obj, "code"), "code is not editable")
}

func TestUpdateAndGetMissingCouponAre404(t *testing.T) {
	srv, _, tm := newEnv(t)
	missing := uuid.New().String()

	resp, _ := do(t, srv, tm, http.MethodGet, "/coupons/"+missing, nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	resp, _ = do(t, srv, tm, http.MethodPatch, "/coupons/"+missing, patchBody(`{"active":true}`))
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestUpdateCouponPreservesOmittedFields is the PATCH-is-not-PUT guarantee.
//
// An admin deactivating one coupon sends only {"active":false}. Before partial
// semantics, that same request silently turned a coupon that expired on
// 2026-01-01 with one use left into a NEVER-EXPIRING, UNLIMITED-USE coupon,
// because every omitted nullable field read as "clear it".
func TestUpdateCouponPreservesOmittedFields(t *testing.T) {
	srv, db, tm := newEnv(t)

	starts := time.Now().UTC().Add(-48 * time.Hour).Truncate(time.Second)
	expires := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second)
	c := seedCoupon(t, db, tm, coupon.NewBuilder("PARTIAL").
		SetDescription("winter promo").
		SetStartsAt(&starts).
		SetExpiresAt(&expires).
		SetMaxUses(ptrU32(1)).
		SetRewards(currencyBundle()))

	resp, out := do(t, srv, tm, http.MethodPatch, "/coupons/"+c.Id().String(), patchBody(`{"active":false}`))
	require.Equal(t, http.StatusOK, resp.StatusCode, string(out))

	// Assert on a fresh GET, not just the PATCH response, so this is about
	// what was STORED rather than what the handler happened to echo back.
	resp, out = do(t, srv, tm, http.MethodGet, "/coupons/"+c.Id().String(), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(out))
	obj := *decodeDoc(t, out).Data.DataObject

	assert.Equal(t, false, attr(t, obj, "active"), "the one field that was sent must change")
	assert.Equal(t, "winter promo", attr(t, obj, "description"), "an omitted description must survive")
	assert.EqualValues(t, 1, attr(t, obj, "maxUses"), "an omitted maxUses must NOT become unlimited")

	expiresRaw, ok := attr(t, obj, "expiresAt").(string)
	require.True(t, ok, "an omitted expiresAt must NOT be cleared")
	gotExpires, err := time.Parse(time.RFC3339, expiresRaw)
	require.NoError(t, err)
	assert.True(t, expires.Equal(gotExpires), "expiresAt = %v, want the stored %v", gotExpires, expires)

	startsRaw, ok := attr(t, obj, "startsAt").(string)
	require.True(t, ok, "an omitted startsAt must NOT be cleared")
	gotStarts, err := time.Parse(time.RFC3339, startsRaw)
	require.NoError(t, err)
	assert.True(t, starts.Equal(gotStarts), "startsAt = %v, want the stored %v", gotStarts, starts)

	rewards, ok := attr(t, obj, "rewards").([]interface{})
	require.True(t, ok, "an omitted rewards bundle must survive")
	require.Len(t, rewards, 1)
	assert.EqualValues(t, 100, rewards[0].(map[string]interface{})["amount"])
}

// TestUpdateCouponClearsExplicitNulls is the other half of partial semantics:
// absence preserves, but an explicit null still clears, so an admin CAN remove
// an expiry or lift a usage cap.
func TestUpdateCouponClearsExplicitNulls(t *testing.T) {
	srv, db, tm := newEnv(t)

	starts := time.Now().UTC().Add(-48 * time.Hour).Truncate(time.Second)
	expires := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second)
	c := seedCoupon(t, db, tm, coupon.NewBuilder("CLEARME").
		SetDescription("temporary").
		SetStartsAt(&starts).
		SetExpiresAt(&expires).
		SetMaxUses(ptrU32(1)).
		SetRewards(currencyBundle()))

	body := patchBody(`{"startsAt":null,"expiresAt":null,"maxUses":null,"description":null}`)
	resp, out := do(t, srv, tm, http.MethodPatch, "/coupons/"+c.Id().String(), body)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(out))

	resp, out = do(t, srv, tm, http.MethodGet, "/coupons/"+c.Id().String(), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(out))
	obj := *decodeDoc(t, out).Data.DataObject

	// These attributes are omitempty on the response, so cleared means absent.
	assert.Nil(t, attr(t, obj, "startsAt"))
	assert.Nil(t, attr(t, obj, "expiresAt"))
	assert.Nil(t, attr(t, obj, "maxUses"))
	assert.Nil(t, attr(t, obj, "description"))
	assert.Equal(t, true, attr(t, obj, "active"), "an omitted active must survive the clears")
}

// TestUpdateCouponRejectsAnExplicitlyEmptiedBundle pins that `rewards: null`
// is NOT a way to make a coupon that grants nothing.
func TestUpdateCouponRejectsAnExplicitlyEmptiedBundle(t *testing.T) {
	srv, db, tm := newEnv(t)
	c := seedCoupon(t, db, tm, coupon.NewBuilder("KEEPREWARDS").SetRewards(currencyBundle()))

	for name, body := range map[string]string{
		"null":  `{"rewards":null}`,
		"empty": `{"rewards":[]}`,
	} {
		t.Run(name, func(t *testing.T) {
			resp, out := do(t, srv, tm, http.MethodPatch, "/coupons/"+c.Id().String(), patchBody(body))
			assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode, string(out))
		})
	}
}

// --- DELETE /coupons/{id} ----------------------------------------------------

func TestDeleteCouponSucceedsThenConflictsOnceRedeemed(t *testing.T) {
	srv, db, tm := newEnv(t)

	clean := seedCoupon(t, db, tm, coupon.NewBuilder("CLEAN").SetRewards(currencyBundle()))
	resp, _ := do(t, srv, tm, http.MethodDelete, "/coupons/"+clean.Id().String(), nil)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	resp, _ = do(t, srv, tm, http.MethodGet, "/coupons/"+clean.Id().String(), nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	redeemed := seedCoupon(t, db, tm, coupon.NewBuilder("REDEEMED").SetRewards(currencyBundle()))
	seedRedemption(t, db, tm, redeemed.Id(), 9001)
	resp, out := do(t, srv, tm, http.MethodDelete, "/coupons/"+redeemed.Id().String(), nil)
	assert.Equal(t, http.StatusConflict, resp.StatusCode, string(out))

	resp, _ = do(t, srv, tm, http.MethodGet, "/coupons/"+redeemed.Id().String(), nil)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "the refused delete must leave the coupon in place")
}

// --- GET /coupons ------------------------------------------------------------

func TestGetCouponsFilters(t *testing.T) {
	srv, db, tm := newEnv(t)
	batchId := uuid.New()
	past := time.Now().Add(-24 * time.Hour)
	future := time.Now().Add(24 * time.Hour)

	seedCoupon(t, db, tm, coupon.NewBuilder("ACTIVEONE").SetBatchId(batchId).SetExpiresAt(&future).SetRewards(currencyBundle()))
	seedCoupon(t, db, tm, coupon.NewBuilder("INACTIVEONE").SetActive(false).SetExpiresAt(&past).SetRewards(currencyBundle()))

	for name, tc := range map[string]struct {
		query string
		want  string
	}{
		"active false":   {"?filter[active]=false", "INACTIVEONE"},
		"active true":    {"?filter[active]=true", "ACTIVEONE"},
		"code":           {"?filter[code]=inactiveone", "INACTIVEONE"},
		"batchId":        {"?filter[batchId]=" + batchId.String(), "ACTIVEONE"},
		"expires before": {"?filter[expiresBefore]=" + time.Now().UTC().Format(time.RFC3339), "INACTIVEONE"},
		"expires after":  {"?filter[expiresAfter]=" + time.Now().UTC().Format(time.RFC3339), "ACTIVEONE"},
	} {
		t.Run(name, func(t *testing.T) {
			resp, out := do(t, srv, tm, http.MethodGet, "/coupons"+tc.query, nil)
			require.Equal(t, http.StatusOK, resp.StatusCode, string(out))
			doc := decodeDoc(t, out)
			require.Len(t, doc.Data.DataArray, 1)
			assert.Equal(t, tc.want, attr(t, doc.Data.DataArray[0], "code"))
			assert.EqualValues(t, 1, doc.Meta["total"], "the total must reflect the filtered scope")
		})
	}

	t.Run("an unparseable filter is 400", func(t *testing.T) {
		resp, _ := do(t, srv, tm, http.MethodGet, "/coupons?filter[active]=maybe", nil)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// --- pagination contract -----------------------------------------------------

// TestListEndpointsHonourThePaginationContract asserts the envelope of
// docs/rest-pagination.md §2 on EVERY list route: meta.total/meta.page,
// first/self/last links, prev omitted on page 1, next omitted on the last
// page, 400 on an out-of-range page[size], and a past-end page returning 200
// with an empty data array and prev pointing at the last real page.
func TestListEndpointsHonourThePaginationContract(t *testing.T) {
	srv, db, tm := newEnv(t)

	// Three coupons, three redemptions of one of them by three accounts, and
	// three batches — one page of two plus a remainder on each route.
	target := seedCoupon(t, db, tm, coupon.NewBuilder("PAGEA").SetRewards(currencyBundle()))
	seedCoupon(t, db, tm, coupon.NewBuilder("PAGEB").SetRewards(currencyBundle()))
	seedCoupon(t, db, tm, coupon.NewBuilder("PAGEC").SetRewards(currencyBundle()))
	for _, accountId := range []uint32{1, 2, 3} {
		seedRedemption(t, db, tm, target.Id(), accountId)
	}
	for i := 0; i < 3; i++ {
		body := marshalBatch(t, batch.RestModel{Count: 1, Length: 8, Prefix: fmt.Sprintf("B%d", i), Rewards: currencyBundle()})
		resp, out := do(t, srv, tm, http.MethodPost, "/coupon-batches", body)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(out))
	}

	for name, path := range map[string]string{
		"coupons":             "/coupons",
		"coupon batches":      "/coupon-batches",
		"coupon redemptions":  "/coupons/" + target.Id().String() + "/redemptions",
		"account redemptions": "/coupon-redemptions?filter[accountId]=1",
	} {
		t.Run(name, func(t *testing.T) {
			sep := "?"
			if strings.Contains(path, "?") {
				sep = "&"
			}

			t.Run("first page of a multi-page result", func(t *testing.T) {
				resp, out := do(t, srv, tm, http.MethodGet, path+sep+"page[number]=1&page[size]=1", nil)
				require.Equal(t, http.StatusOK, resp.StatusCode, string(out))
				doc := decodeDoc(t, out)
				require.Len(t, doc.Data.DataArray, 1)
				require.NotNil(t, doc.Meta)
				page := doc.Meta["page"].(map[string]interface{})
				assert.EqualValues(t, 1, page["number"])
				assert.EqualValues(t, 1, page["size"])
				assert.EqualValues(t, doc.Meta["total"], page["last"], "size 1 makes last == total")
				require.NotNil(t, doc.Links)
				assert.Contains(t, doc.Links, "self")
				assert.Contains(t, doc.Links, "first")
				assert.Contains(t, doc.Links, "last")
				assert.NotContains(t, doc.Links, "prev", "prev is omitted on page 1")
			})

			t.Run("last page omits next", func(t *testing.T) {
				resp, out := do(t, srv, tm, http.MethodGet, path+sep+"page[number]=1&page[size]=250", nil)
				require.Equal(t, http.StatusOK, resp.StatusCode, string(out))
				doc := decodeDoc(t, out)
				assert.NotContains(t, doc.Links, "next")
			})

			t.Run("page size over the max is 400", func(t *testing.T) {
				resp, _ := do(t, srv, tm, http.MethodGet, path+sep+"page[size]=251", nil)
				assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			})

			t.Run("past the end is an empty 200 with prev at last", func(t *testing.T) {
				resp, out := do(t, srv, tm, http.MethodGet, path+sep+"page[number]=99&page[size]=1", nil)
				require.Equal(t, http.StatusOK, resp.StatusCode, string(out))
				doc := decodeDoc(t, out)
				assert.Len(t, doc.Data.DataArray, 0)
				assert.Contains(t, doc.Links, "prev")
				assert.NotContains(t, doc.Links, "next")
			})
		})
	}
}

// --- redemption scoping ------------------------------------------------------

func TestRedemptionRoutesReturnOnlyTheirScope(t *testing.T) {
	srv, db, tm := newEnv(t)

	a := seedCoupon(t, db, tm, coupon.NewBuilder("SCOPEA").SetRewards(currencyBundle()))
	b := seedCoupon(t, db, tm, coupon.NewBuilder("SCOPEB").SetRewards(currencyBundle()))
	seedRedemption(t, db, tm, a.Id(), 111)
	seedRedemption(t, db, tm, a.Id(), 222)
	seedRedemption(t, db, tm, b.Id(), 111)

	t.Run("by coupon", func(t *testing.T) {
		resp, out := do(t, srv, tm, http.MethodGet, "/coupons/"+a.Id().String()+"/redemptions", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(out))
		doc := decodeDoc(t, out)
		assert.EqualValues(t, 2, doc.Meta["total"])
		for _, d := range doc.Data.DataArray {
			assert.Equal(t, a.Id().String(), attr(t, d, "couponId"))
		}
	})

	t.Run("by account", func(t *testing.T) {
		resp, out := do(t, srv, tm, http.MethodGet, "/coupon-redemptions?filter[accountId]=222", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(out))
		doc := decodeDoc(t, out)
		require.Len(t, doc.Data.DataArray, 1)
		assert.EqualValues(t, 222, attr(t, doc.Data.DataArray[0], "accountId"))
	})

	t.Run("a missing accountId filter is 400", func(t *testing.T) {
		resp, _ := do(t, srv, tm, http.MethodGet, "/coupon-redemptions", nil)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// --- POST /coupon-batches ----------------------------------------------------

// TestCreateBatchGenerates500UniqueCodes is a THROUGHPUT AND UNIQUENESS test,
// not a concurrency test: the harness caps the pool at one connection, so this
// is one sequential generation. What it pins is that a 500-code request yields
// 500 DISTINCT codes and a batch whose generatedCount is 500 — the guarantee
// the collision RETRY exists to provide.
func TestCreateBatchGenerates500UniqueCodes(t *testing.T) {
	srv, _, tm := newEnv(t)

	body := marshalBatch(t, batch.RestModel{
		Count: 500, Prefix: "sum24-", Length: 8, Description: "summer", Rewards: currencyBundle(),
	})
	resp, out := do(t, srv, tm, http.MethodPost, "/coupon-batches", body)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(out))

	obj := *decodeDoc(t, out).Data.DataObject
	assert.EqualValues(t, 500, attr(t, obj, "requestedCount"))
	assert.EqualValues(t, 500, attr(t, obj, "generatedCount"))
	assert.EqualValues(t, 0, attr(t, obj, "redeemedCount"))

	raw, ok := attr(t, obj, "codes").([]interface{})
	require.True(t, ok, "the POST response carries the plaintext codes")
	require.Len(t, raw, 500)
	seen := map[string]bool{}
	for _, v := range raw {
		code := v.(string)
		assert.True(t, strings.HasPrefix(code, "SUM24-"), "the prefix is normalized like any other code")
		assert.False(t, seen[code], "duplicate code %q in one batch", code)
		seen[code] = true
	}

	batchId := obj.ID
	t.Run("the coupons are listable by batch", func(t *testing.T) {
		resp, out := do(t, srv, tm, http.MethodGet, "/coupons?filter[batchId]="+batchId+"&page[size]=1", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(out))
		assert.EqualValues(t, 500, decodeDoc(t, out).Meta["total"])
	})

	t.Run("a later GET does not re-serve the codes", func(t *testing.T) {
		resp, out := do(t, srv, tm, http.MethodGet, "/coupon-batches/"+batchId, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(out))
		assert.Nil(t, attr(t, *decodeDoc(t, out).Data.DataObject, "codes"))
	})
}

func TestCreateBatchRejectsAnOverLongPrefixAndLengthWith422(t *testing.T) {
	srv, _, tm := newEnv(t)

	// 30-character prefix plus a length of 8 is 38 > the 32-character column.
	body := marshalBatch(t, batch.RestModel{
		Count: 1, Prefix: strings.Repeat("P", 30), Length: 8, Rewards: currencyBundle(),
	})
	resp, out := do(t, srv, tm, http.MethodPost, "/coupon-batches", body)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode, string(out))

	t.Run("a zero count is 422", func(t *testing.T) {
		body := marshalBatch(t, batch.RestModel{Count: 0, Length: 8, Rewards: currencyBundle()})
		resp, out := do(t, srv, tm, http.MethodPost, "/coupon-batches", body)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode, string(out))
	})

	t.Run("an empty bundle is 422", func(t *testing.T) {
		body := marshalBatch(t, batch.RestModel{Count: 1, Length: 8})
		resp, out := do(t, srv, tm, http.MethodPost, "/coupon-batches", body)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode, string(out))
	})
}

func TestGetBatchReportsRedeemedCount(t *testing.T) {
	srv, db, tm := newEnv(t)

	body := marshalBatch(t, batch.RestModel{Count: 3, Length: 8, Rewards: currencyBundle()})
	resp, out := do(t, srv, tm, http.MethodPost, "/coupon-batches", body)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(out))
	batchId := decodeDoc(t, out).Data.DataObject.ID

	// Redeem one of the batch's coupons by inserting the audit row directly.
	var e coupon.Entity
	require.NoError(t, db.Where("batch_id = ?", batchId).First(&e).Error)
	seedRedemption(t, db, tm, e.Id, 7)

	resp, out = do(t, srv, tm, http.MethodGet, "/coupon-batches/"+batchId, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(out))
	obj := *decodeDoc(t, out).Data.DataObject
	assert.EqualValues(t, 3, attr(t, obj, "generatedCount"))
	assert.EqualValues(t, 1, attr(t, obj, "redeemedCount"))
}

func TestGetMissingBatchIs404(t *testing.T) {
	srv, _, tm := newEnv(t)
	resp, _ := do(t, srv, tm, http.MethodGet, "/coupon-batches/"+uuid.New().String(), nil)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// --- multi-tenancy -----------------------------------------------------------

// TestAnotherTenantCanNeitherReadNorMutateTheseCoupons is the multi-tenancy
// guarantee, asserted at the REQUEST level rather than the provider level: the
// intruder's requests go through the same router, the same handlers and the
// same processors as the owner's, differing only in the tenant headers.
func TestAnotherTenantCanNeitherReadNorMutateTheseCoupons(t *testing.T) {
	srv, db, owner := newEnv(t)
	intruder, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)

	c := seedCoupon(t, db, owner, coupon.NewBuilder("OWNED").SetDescription("owner's").SetRewards(currencyBundle()))
	seedRedemption(t, db, owner, c.Id(), 555)

	ownerBatch := marshalBatch(t, batch.RestModel{Count: 2, Length: 8, Rewards: currencyBundle()})
	resp, out := do(t, srv, owner, http.MethodPost, "/coupon-batches", ownerBatch)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(out))
	ownerBatchId := decodeDoc(t, out).Data.DataObject.ID

	id := c.Id().String()

	t.Run("cannot read the coupon", func(t *testing.T) {
		resp, _ := do(t, srv, intruder, http.MethodGet, "/coupons/"+id, nil)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("cannot list the coupons", func(t *testing.T) {
		resp, out := do(t, srv, intruder, http.MethodGet, "/coupons", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		doc := decodeDoc(t, out)
		assert.Len(t, doc.Data.DataArray, 0)
		assert.EqualValues(t, 0, doc.Meta["total"])
	})

	t.Run("cannot update the coupon", func(t *testing.T) {
		resp, _ := do(t, srv, intruder, http.MethodPatch, "/coupons/"+id, patchBody(`{"description":"stolen","active":false}`))
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("cannot delete the coupon", func(t *testing.T) {
		resp, _ := do(t, srv, intruder, http.MethodDelete, "/coupons/"+id, nil)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("cannot read the redemptions", func(t *testing.T) {
		resp, out := do(t, srv, intruder, http.MethodGet, "/coupons/"+id+"/redemptions", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.EqualValues(t, 0, decodeDoc(t, out).Meta["total"])

		resp, out = do(t, srv, intruder, http.MethodGet, "/coupon-redemptions?filter[accountId]=555", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.EqualValues(t, 0, decodeDoc(t, out).Meta["total"])
	})

	t.Run("cannot read the batch", func(t *testing.T) {
		resp, _ := do(t, srv, intruder, http.MethodGet, "/coupon-batches/"+ownerBatchId, nil)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		resp, out := do(t, srv, intruder, http.MethodGet, "/coupon-batches", nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.EqualValues(t, 0, decodeDoc(t, out).Meta["total"])
	})

	t.Run("the owner's rows are untouched", func(t *testing.T) {
		resp, out := do(t, srv, owner, http.MethodGet, "/coupons/"+id, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode, string(out))
		assert.Equal(t, "owner's", attr(t, *decodeDoc(t, out).Data.DataObject, "description"))
	})
}

// --- no redeem endpoint ------------------------------------------------------

// TestThereIsNoRedeemEndpoint pins the security decision this whole surface is
// shaped by: a REST redeem would be an unauthenticated reward faucet, so the
// packet path is the only trigger. If someone adds one, this fails.
func TestThereIsNoRedeemEndpoint(t *testing.T) {
	srv, db, tm := newEnv(t)
	c := seedCoupon(t, db, tm, coupon.NewBuilder("NOFAUCET").SetRewards(currencyBundle()))

	for _, path := range []string{
		"/coupons/" + c.Id().String() + "/redeem",
		"/coupons/redeem",
		"/coupon-redemptions",
	} {
		for _, method := range []string{http.MethodPost, http.MethodPut} {
			resp, _ := do(t, srv, tm, method, path, marshalCoupon(t, coupon.RestModel{Code: "NOFAUCET"}))
			assert.NotContains(t, []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent},
				resp.StatusCode, "%s %s must not succeed", method, path)
		}
	}
}
