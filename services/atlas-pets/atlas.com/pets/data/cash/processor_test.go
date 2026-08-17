package cash_test

import (
	"atlas-pets/data/cash"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
)

func TestGetByIdReadsLife(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"id":"5180000","type":"cash_items","attributes":{"life":90}}}`))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	l, _ := test.NewNullLogger()
	m, err := cash.NewProcessor(l, context.Background()).GetById(5180000)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if m.Life() != 90 {
		t.Errorf("Life = %d, want 90", m.Life())
	}
}

func TestGetByIdLifeAbsentIsZero(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"id":"5180000","type":"cash_items","attributes":{}}}`))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	l, _ := test.NewNullLogger()
	m, err := cash.NewProcessor(l, context.Background()).GetById(5180000)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if m.Life() != 0 {
		t.Errorf("Life = %d, want 0", m.Life())
	}
}
