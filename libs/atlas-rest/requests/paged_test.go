package requests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus/hooks/test"
)

type pagedFixture struct {
	Id   uint32 `json:"-"`
	Name string `json:"name"`
}

func (f pagedFixture) GetName() string { return "fixtures" }
func (f pagedFixture) GetID() string   { return strconv.Itoa(int(f.Id)) }
func (f *pagedFixture) SetID(id string) error {
	v, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	f.Id = uint32(v)
	return nil
}
func (f *pagedFixture) SetToOneReferenceID(_, _ string) error            { return nil }
func (f *pagedFixture) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

func extractFixture(f pagedFixture) (uint32, error) { return f.Id, nil }

// pageDoc renders a JSON:API page. ids come from [from, to). last/total per args.
// Each resource carries a relationships block to pin the api2go stub requirement.
func pageDoc(from, to, total, number, size, last int) string {
	data := ""
	for i := from; i < to; i++ {
		if data != "" {
			data += ","
		}
		data += fmt.Sprintf(`{"id":"%d","type":"fixtures","attributes":{"name":"n%d"},"relationships":{"tags":{"data":[{"id":"1","type":"tags"}]}}}`, i, i)
	}
	return fmt.Sprintf(`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`, data, total, number, size, last)
}

func servePages(t *testing.T, totalItems, pageSize int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		number, _ := strconv.Atoi(q.Get("page[number]"))
		size, _ := strconv.Atoi(q.Get("page[size]"))
		if number < 1 || size < 1 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		last := (totalItems + size - 1) / size
		if last < 1 {
			last = 1
		}
		from := (number-1)*size + 1
		to := from + size
		if from > totalItems {
			from, to = 1, 1 // empty page
		} else if to > totalItems+1 {
			to = totalItems + 1
		}
		_, _ = w.Write([]byte(pageDoc(from, to, totalItems, number, size, last)))
	}))
}

func TestPagedGetRequestDecodesItemsAndMeta(t *testing.T) {
	srv := servePages(t, 5, 2)
	defer srv.Close()
	l, _ := test.NewNullLogger()
	resp, err := PagedGetRequest[pagedFixture](srv.URL+"/fixtures", model.Page{Number: 2, Size: 2})(l, context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 2 || resp.Data[0].Id != 3 {
		t.Fatalf("data: %+v", resp.Data)
	}
	if resp.Meta == nil || resp.Meta.Total != 5 || resp.Meta.Page.Last != 3 {
		t.Fatalf("meta: %+v", resp.Meta)
	}
}

func TestPagedGetRequestPreservesExistingQuery(t *testing.T) {
	var gotFilter string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotFilter = r.URL.Query().Get("filter[name]")
		_, _ = w.Write([]byte(pageDoc(1, 1, 0, 1, 50, 1)))
	}))
	defer srv.Close()
	l, _ := test.NewNullLogger()
	_, err := PagedGetRequest[pagedFixture](srv.URL+"/fixtures?filter[name]=bob", model.Page{Number: 1, Size: 50})(l, context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotFilter != "bob" {
		t.Fatalf("existing query param lost: %q", gotFilter)
	}
}

func TestDrainProviderMultiPage(t *testing.T) {
	srv := servePages(t, 5, 2)
	defer srv.Close()
	l, _ := test.NewNullLogger()
	ms, err := DrainProvider[pagedFixture, uint32](l, context.Background())(srv.URL+"/fixtures", 2, extractFixture, model.Filters[uint32]())()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 5 || ms[0] != 1 || ms[4] != 5 {
		t.Fatalf("drained: %v", ms)
	}
}

func TestDrainProviderSinglePage(t *testing.T) {
	srv := servePages(t, 3, 250)
	defer srv.Close()
	l, _ := test.NewNullLogger()
	ms, err := DrainProvider[pagedFixture, uint32](l, context.Background())(srv.URL+"/fixtures", 250, extractFixture, model.Filters[uint32]())()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 3 {
		t.Fatalf("drained: %v", ms)
	}
}

func TestDrainProviderEmptyCollection(t *testing.T) {
	srv := servePages(t, 0, 50)
	defer srv.Close()
	l, _ := test.NewNullLogger()
	ms, err := DrainProvider[pagedFixture, uint32](l, context.Background())(srv.URL+"/fixtures", 50, extractFixture, model.Filters[uint32]())()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 0 {
		t.Fatalf("drained: %v", ms)
	}
}

func TestDrainProviderNoEnvelopeCompat(t *testing.T) {
	// Unconverted server: plain document, no meta. The single response IS
	// the complete collection.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"1","type":"fixtures","attributes":{"name":"a"},"relationships":{"tags":{"data":[]}}},{"id":"2","type":"fixtures","attributes":{"name":"b"},"relationships":{"tags":{"data":[]}}}]}`))
	}))
	defer srv.Close()
	l, _ := test.NewNullLogger()
	ms, err := DrainProvider[pagedFixture, uint32](l, context.Background())(srv.URL+"/fixtures", 50, extractFixture, model.Filters[uint32]())()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 2 {
		t.Fatalf("compat drain: %v", ms)
	}
}

func TestDrainProviderWarnsPast20Pages(t *testing.T) {
	srv := servePages(t, 45, 2) // 23 pages
	defer srv.Close()
	l, hook := test.NewNullLogger()
	ms, err := DrainProvider[pagedFixture, uint32](l, context.Background())(srv.URL+"/fixtures", 2, extractFixture, model.Filters[uint32]())()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 45 {
		t.Fatalf("drained %d", len(ms))
	}
	warned := false
	for _, e := range hook.AllEntries() {
		if e.Level.String() == "warning" {
			warned = true
		}
	}
	if !warned {
		t.Fatal("expected a warning for a >20-page drain")
	}
}

func TestPagedProviderErrorMapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	l, _ := test.NewNullLogger()
	_, err := PagedProvider[pagedFixture, uint32](l, context.Background())(srv.URL+"/fixtures", model.Page{Number: 1, Size: 50}, extractFixture)()
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestPagedProviderReturnsPagedModel(t *testing.T) {
	srv := servePages(t, 5, 2)
	defer srv.Close()
	l, _ := test.NewNullLogger()
	p, err := PagedProvider[pagedFixture, uint32](l, context.Background())(srv.URL+"/fixtures", model.Page{Number: 1, Size: 2}, extractFixture)()
	if err != nil {
		t.Fatal(err)
	}
	if p.Total != 5 || p.Page.Number != 1 || len(p.Items) != 2 {
		t.Fatalf("%+v", p)
	}
}
