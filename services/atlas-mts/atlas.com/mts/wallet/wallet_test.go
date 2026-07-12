package wallet

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

// TestEnsureWallet_ExistsNoCreate proves a wallet that already exists (GET 200) is
// left untouched — no create POST is issued.
func TestEnsureWallet_ExistsNoCreate(t *testing.T) {
	var gotPost bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			gotPost = true
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"type":"wallets","id":"00000000-0000-0000-0000-000000000000","attributes":{"accountId":42,"credit":0,"points":0,"prepaid":0}}}`))
	}))
	defer srv.Close()
	t.Setenv("CASHSHOP_SERVICE_URL", srv.URL+"/")

	if err := NewProcessor(logrus.New(), context.Background()).EnsureWallet(42, 0, 0, 0); err != nil {
		t.Fatalf("EnsureWallet returned error for existing wallet: %v", err)
	}
	if gotPost {
		t.Fatalf("EnsureWallet issued a create POST for an already-existing wallet")
	}
}

// TestEnsureWallet_MissingCreates proves a missing wallet (GET 404) triggers a
// create POST — the seed-seller provisioning path.
func TestEnsureWallet_MissingCreates(t *testing.T) {
	var gotPost bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			gotPost = true
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"type":"wallets","id":"00000000-0000-0000-0000-000000000000","attributes":{"accountId":42}}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	t.Setenv("CASHSHOP_SERVICE_URL", srv.URL+"/")

	if err := NewProcessor(logrus.New(), context.Background()).EnsureWallet(42, 0, 0, 0); err != nil {
		t.Fatalf("EnsureWallet returned error creating a missing wallet: %v", err)
	}
	if !gotPost {
		t.Fatalf("EnsureWallet did not issue a create POST for a missing wallet")
	}
}
