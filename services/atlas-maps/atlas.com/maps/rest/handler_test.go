package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus/hooks/test"
)

func TestParseWorldId_Valid(t *testing.T) {
	l, _ := test.NewNullLogger()

	tests := []struct {
		name     string
		worldId  string
		expected byte
	}{
		{"zero", "0", 0},
		{"one", "1", 1},
		{"mid range", "128", 128},
		{"max byte", "255", 255},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedWorldId byte
			handler := ParseWorldId(l, func(worldId byte) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					capturedWorldId = worldId
					w.WriteHeader(http.StatusOK)
				}
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req = mux.SetURLVars(req, map[string]string{"worldId": tt.worldId})
			rr := httptest.NewRecorder()

			handler(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}
			if capturedWorldId != tt.expected {
				t.Errorf("Expected worldId %d, got %d", tt.expected, capturedWorldId)
			}
		})
	}
}

func TestParseWorldId_Invalid(t *testing.T) {
	l, _ := test.NewNullLogger()

	tests := []struct {
		name    string
		worldId string
	}{
		{"non-numeric", "invalid"},
		{"empty", ""},
		{"float", "1.5"},
		{"special chars", "abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			handler := ParseWorldId(l, func(worldId byte) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					handlerCalled = true
					w.WriteHeader(http.StatusOK)
				}
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req = mux.SetURLVars(req, map[string]string{"worldId": tt.worldId})
			rr := httptest.NewRecorder()

			handler(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", rr.Code)
			}
			if handlerCalled {
				t.Error("Handler should not have been called for invalid input")
			}
		})
	}
}

func TestParseWorldId_MissingVar(t *testing.T) {
	l, _ := test.NewNullLogger()

	handlerCalled := false
	handler := ParseWorldId(l, func(worldId byte) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No URL vars set
	rr := httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
	if handlerCalled {
		t.Error("Handler should not have been called for missing var")
	}
}

func TestParseChannelId_Valid(t *testing.T) {
	l, _ := test.NewNullLogger()

	tests := []struct {
		name      string
		channelId string
		expected  byte
	}{
		{"zero", "0", 0},
		{"one", "1", 1},
		{"mid range", "10", 10},
		{"max byte", "255", 255},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedChannelId byte
			handler := ParseChannelId(l, func(channelId byte) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					capturedChannelId = channelId
					w.WriteHeader(http.StatusOK)
				}
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req = mux.SetURLVars(req, map[string]string{"channelId": tt.channelId})
			rr := httptest.NewRecorder()

			handler(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}
			if capturedChannelId != tt.expected {
				t.Errorf("Expected channelId %d, got %d", tt.expected, capturedChannelId)
			}
		})
	}
}

func TestParseChannelId_Invalid(t *testing.T) {
	l, _ := test.NewNullLogger()

	tests := []struct {
		name      string
		channelId string
	}{
		{"non-numeric", "invalid"},
		{"empty", ""},
		{"float", "1.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			handler := ParseChannelId(l, func(channelId byte) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					handlerCalled = true
					w.WriteHeader(http.StatusOK)
				}
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req = mux.SetURLVars(req, map[string]string{"channelId": tt.channelId})
			rr := httptest.NewRecorder()

			handler(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", rr.Code)
			}
			if handlerCalled {
				t.Error("Handler should not have been called for invalid input")
			}
		})
	}
}

func TestParseChannelId_MissingVar(t *testing.T) {
	l, _ := test.NewNullLogger()

	handlerCalled := false
	handler := ParseChannelId(l, func(channelId byte) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
	if handlerCalled {
		t.Error("Handler should not have been called for missing var")
	}
}

func TestParseMapId_Valid(t *testing.T) {
	l, _ := test.NewNullLogger()

	tests := []struct {
		name     string
		mapId    string
		expected uint32
	}{
		{"zero", "0", 0},
		{"typical map id", "100000000", 100000000},
		{"another map", "200000000", 200000000},
		{"small value", "1", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedMapId uint32
			handler := ParseMapId(l, func(mapId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					capturedMapId = mapId
					w.WriteHeader(http.StatusOK)
				}
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req = mux.SetURLVars(req, map[string]string{"mapId": tt.mapId})
			rr := httptest.NewRecorder()

			handler(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}
			if capturedMapId != tt.expected {
				t.Errorf("Expected mapId %d, got %d", tt.expected, capturedMapId)
			}
		})
	}
}

func TestParseMapId_Invalid(t *testing.T) {
	l, _ := test.NewNullLogger()

	tests := []struct {
		name  string
		mapId string
	}{
		{"non-numeric", "invalid"},
		{"empty", ""},
		{"float", "100000000.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			handler := ParseMapId(l, func(mapId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					handlerCalled = true
					w.WriteHeader(http.StatusOK)
				}
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req = mux.SetURLVars(req, map[string]string{"mapId": tt.mapId})
			rr := httptest.NewRecorder()

			handler(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", rr.Code)
			}
			if handlerCalled {
				t.Error("Handler should not have been called for invalid input")
			}
		})
	}
}

func TestParseMapId_MissingVar(t *testing.T) {
	l, _ := test.NewNullLogger()

	handlerCalled := false
	handler := ParseMapId(l, func(mapId uint32) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
	if handlerCalled {
		t.Error("Handler should not have been called for missing var")
	}
}

func TestHandlerDependency_Logger(t *testing.T) {
	l, _ := test.NewNullLogger()
	d := HandlerDependency{l: l}

	if d.Logger() != l {
		t.Error("Logger() should return the stored logger")
	}
}

func TestHandlerDependency_Context(t *testing.T) {
	l, _ := test.NewNullLogger()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := req.Context()
	d := HandlerDependency{l: l, ctx: ctx}

	if d.Context() != ctx {
		t.Error("Context() should return the stored context")
	}
}
