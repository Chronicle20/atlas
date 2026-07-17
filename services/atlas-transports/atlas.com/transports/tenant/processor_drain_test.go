package tenant_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-transports/tenant"

	atlastenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// tenantDoc renders a JSON:API document for tenants [from, to]. Each tenant
// gets a distinct uuid derived from its index so the test can assert
// presence of a page-2-only item.
func tenantDoc(from, to int, total, number, size, last int) string {
	var b strings.Builder
	for i := from; i <= to; i++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		id := fmt.Sprintf("00000000-0000-0000-0000-%012d", i)
		b.WriteString(fmt.Sprintf(
			`{"id":"%s","type":"tenants","attributes":{"name":"tenant-%d","region":"GMS","majorVersion":83,"minorVersion":1}}`,
			id, i,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestAllProviderDrainsBeyondOnePage proves tenant.Processor.AllProvider
// (via requests.DrainProvider) fetches every page of the tenants list
// rather than stopping after the first response. atlas-tenants' GET
// /tenants is now paginated (task-117); main.go's startup per-tenant route
// config load is a genuine semantic-all consumer that must see every
// tenant. The fixture server serves 260 tenants across two pages of 250 -
// only a genuine drain picks up the tenant on page 2.
func TestAllProviderDrainsBeyondOnePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(tenantDoc(251, 260, 260, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(tenantDoc(1, 250, 260, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("TENANTS_SERVICE_URL", srv.URL+"/")

	ten, err := atlastenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := atlastenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	ts, err := tenant.NewProcessor(l, ctx).GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 260 {
		t.Fatalf("expected 260 tenants (full drain), got %d; a single-fetch implementation would return 250", len(ts))
	}

	wantLast := fmt.Sprintf("00000000-0000-0000-0000-%012d", 260)
	foundLast := false
	for _, tm := range ts {
		if tm.Id().String() == wantLast {
			foundLast = true
		}
	}
	if !foundLast {
		t.Fatal("tenant 260 (page 2) must be present; single-fetch impl would miss it")
	}
}
