package wallet

import (
	"atlas-mts/rest"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type testServerInfo struct{}

func (t testServerInfo) GetBaseURL() string { return "http://localhost:8080" }
func (t testServerInfo) GetPrefix() string  { return "/api" }

// stubBalanceReader returns canned two-bucket balances, capturing the accountId
// it was asked for. An err field lets a test exercise the 500 path.
type stubBalanceReader struct {
	prepaid uint32
	points  uint32
	err     error
	seen    uint32
}

func (s *stubBalanceReader) Balance(accountId uint32) (uint32, uint32, error) {
	s.seen = accountId
	if s.err != nil {
		return 0, 0, s.err
	}
	return s.prepaid, s.points, nil
}

func newWalletServer(t *testing.T, rf ReaderFactory) *httptest.Server {
	t.Helper()
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	router := mux.NewRouter()
	// db is unused by the wallet read (the read is an outbound cash-shop REST
	// call), but the resource initializer threads it for signature parity.
	initResource(testServerInfo{}, rf)(nil)(router, l)
	return httptest.NewServer(router)
}

func withTenant(t *testing.T, method, url string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("TENANT_ID", "00000000-0000-0000-0000-000000000001")
	req.Header.Set("REGION", "GMS")
	req.Header.Set("MAJOR_VERSION", "83")
	req.Header.Set("MINOR_VERSION", "1")
	return req
}

// TestGetWallet_TwoBuckets asserts the wallet read returns a JSON:API "wallets"
// resource carrying the account's two MTS buckets (prepaid + points), and that
// the handler reads the wallet for the path's accountId.
func TestGetWallet_TwoBuckets(t *testing.T) {
	stub := &stubBalanceReader{prepaid: 12345, points: 678}
	rf := func(_ *rest.HandlerDependency) BalanceReader { return stub }

	srv := newWalletServer(t, rf)
	defer srv.Close()
	client := &http.Client{}

	resp, err := client.Do(withTenant(t, http.MethodGet, fmt.Sprintf("%s/accounts/4242/mts/wallet", srv.URL)))
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var env struct {
		Data struct {
			Type       string `json:"type"`
			Id         string `json:"id"`
			Attributes struct {
				Prepaid uint32 `json:"prepaid"`
				Points  uint32 `json:"points"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Data.Type != "wallets" {
		t.Errorf("type = %q, want wallets", env.Data.Type)
	}
	if env.Data.Id != "4242" {
		t.Errorf("id = %q, want 4242", env.Data.Id)
	}
	if env.Data.Attributes.Prepaid != 12345 {
		t.Errorf("prepaid = %d, want 12345", env.Data.Attributes.Prepaid)
	}
	if env.Data.Attributes.Points != 678 {
		t.Errorf("points = %d, want 678", env.Data.Attributes.Points)
	}
	if stub.seen != 4242 {
		t.Errorf("reader asked for account %d, want 4242", stub.seen)
	}
}

// TestGetWallet_ReadError asserts a failed wallet read surfaces as a 500, not a
// panic or a partial body.
func TestGetWallet_ReadError(t *testing.T) {
	stub := &stubBalanceReader{err: fmt.Errorf("cash-shop unreachable")}
	rf := func(_ *rest.HandlerDependency) BalanceReader { return stub }

	srv := newWalletServer(t, rf)
	defer srv.Close()
	client := &http.Client{}

	resp, err := client.Do(withTenant(t, http.MethodGet, fmt.Sprintf("%s/accounts/4242/mts/wallet", srv.URL)))
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
}
