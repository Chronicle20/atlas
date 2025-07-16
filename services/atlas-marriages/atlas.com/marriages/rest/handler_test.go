package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

// testServerInfo implements jsonapi.ServerInformation for testing
type testServerInfo struct{}

func (t testServerInfo) GetVersion() string  { return "1.0.0" }
func (t testServerInfo) GetURI() string      { return "/api/mas/" }
func (t testServerInfo) GetPrefix() string   { return "/api/mas/" }
func (t testServerInfo) GetBaseURL() string  { return "http://localhost:8080" }

func TestHandlerDependency_Logger(t *testing.T) {
	logger := logrus.New()
	ctx := context.Background()
	
	hd := HandlerDependency{
		l:   logger,
		ctx: ctx,
	}
	
	if hd.Logger() != logger {
		t.Error("Logger() did not return the expected logger")
	}
}

func TestHandlerDependency_Context(t *testing.T) {
	logger := logrus.New()
	ctx := context.Background()
	
	hd := HandlerDependency{
		l:   logger,
		ctx: ctx,
	}
	
	if hd.Context() != ctx {
		t.Error("Context() did not return the expected context")
	}
}

func TestHandlerContext_ServerInformation(t *testing.T) {
	si := &testServerInfo{}
	
	hc := HandlerContext{
		si: si,
	}
	
	if hc.ServerInformation() != si {
		t.Error("ServerInformation() did not return the expected server information")
	}
}

type TestRequest struct {
	Data TestData `json:"data"`
}

type TestData struct {
	Type       string         `json:"type"`
	Attributes TestAttributes `json:"attributes"`
}

type TestAttributes struct {
	Name string `json:"name"`
}

func TestParseInput_Success(t *testing.T) {
	logger := logrus.New()
	ctx := context.Background()
	
	hd := &HandlerDependency{
		l:   logger,
		ctx: ctx,
	}
	
	si := &testServerInfo{}
	
	hc := &HandlerContext{
		si: si,
	}
	
	var receivedModel TestRequest
	testHandler := func(d *HandlerDependency, c *HandlerContext, model TestRequest) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			receivedModel = model
			w.WriteHeader(http.StatusOK)
		}
	}
	
	jsonBody := `{"data": {"type": "test", "attributes": {"name": "test name"}}}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	
	handler := ParseInput[TestRequest](hd, hc, testHandler)
	handler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	if receivedModel.Data.Attributes.Name != "test name" {
		t.Errorf("Expected name 'test name', got '%s'", receivedModel.Data.Attributes.Name)
	}
}

func TestParseInput_InvalidJSON(t *testing.T) {
	logger := logrus.New()
	ctx := context.Background()
	
	hd := &HandlerDependency{
		l:   logger,
		ctx: ctx,
	}
	
	si := &testServerInfo{}
	
	hc := &HandlerContext{
		si: si,
	}
	
	testHandler := func(d *HandlerDependency, c *HandlerContext, model TestRequest) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}
	}
	
	jsonBody := `{"invalid": json}`
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	
	handler := ParseInput[TestRequest](hd, hc, testHandler)
	handler(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestRegisterHandler_ReturnsFunction(t *testing.T) {
	logger := logrus.New()
	
	registrar := RegisterHandler(logger)
	if registrar == nil {
		t.Error("RegisterHandler() returned nil")
	}
	
	si := &testServerInfo{}
	
	siFunc := registrar(si)
	if siFunc == nil {
		t.Error("RegisterHandler()(si) returned nil")
	}
	
	testHandler := func(d *HandlerDependency, c *HandlerContext) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}
	}
	
	handlerFunc := siFunc("test", testHandler)
	if handlerFunc == nil {
		t.Error("RegisterHandler()(si)(name, handler) returned nil")
	}
}

func TestRegisterInputHandler_ReturnsFunction(t *testing.T) {
	logger := logrus.New()
	
	registrar := RegisterInputHandler[TestRequest](logger)
	if registrar == nil {
		t.Error("RegisterInputHandler() returned nil")
	}
	
	si := &testServerInfo{}
	
	siFunc := registrar(si)
	if siFunc == nil {
		t.Error("RegisterInputHandler()(si) returned nil")
	}
	
	testHandler := func(d *HandlerDependency, c *HandlerContext, model TestRequest) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}
	}
	
	handlerFunc := siFunc("test", testHandler)
	if handlerFunc == nil {
		t.Error("RegisterInputHandler()(si)(name, handler) returned nil")
	}
}