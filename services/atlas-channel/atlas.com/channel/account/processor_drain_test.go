package account_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"atlas-channel/account"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
)

// accountDoc renders a JSON:API document for accounts [from, to]. loggedIn
// is 1 for ids present in loggedInIds, 0 otherwise. meta describes the
// current page/total so requests.DrainProvider can decide whether to keep
// paging.
func accountDoc(from, to int, loggedInIds map[int]bool, total, number, size, last int) string {
	var b strings.Builder
	for id := from; id <= to; id++ {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		loggedIn := 0
		if loggedInIds[id] {
			loggedIn = 1
		}
		b.WriteString(fmt.Sprintf(
			`{"id":"%d","type":"accounts","attributes":{"name":"account%d","loggedIn":%d}}`,
			id, id, loggedIn,
		))
	}
	return fmt.Sprintf(
		`{"data":[%s],"meta":{"total":%d,"page":{"number":%d,"size":%d,"last":%d}}}`,
		b.String(), total, number, size, last,
	)
}

// TestInitializeRegistrySeedsAcrossPages proves InitializeRegistry drains
// every page of the accounts collection rather than stopping after the
// first response. The fixture server serves 300 accounts across two pages
// of 250; only a genuine drain picks up id 300, which lives on page 2.
func TestInitializeRegistrySeedsAcrossPages(t *testing.T) {
	loggedInIds := map[int]bool{1: true, 300: true}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		number, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if number == 2 {
			_, _ = w.Write([]byte(accountDoc(251, 300, loggedInIds, 300, 2, 250, 2)))
			return
		}
		_, _ = w.Write([]byte(accountDoc(1, 250, loggedInIds, 300, 1, 250, 2)))
	}))
	defer srv.Close()
	t.Setenv("ACCOUNTS_SERVICE_URL", srv.URL+"/")

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	if err := account.NewProcessor(l, ctx).InitializeRegistry(); err != nil {
		t.Fatal(err)
	}

	p := account.NewProcessor(l, ctx)
	if !p.IsLoggedIn(1) {
		t.Fatal("account 1 (page 1) must be logged in")
	}
	if !p.IsLoggedIn(300) {
		t.Fatal("account 300 (page 2) must be logged in; single-fetch impl would miss it")
	}
	if p.IsLoggedIn(2) {
		t.Fatal("account 2 is logged out and must not report logged in")
	}
}
