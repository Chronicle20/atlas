package server

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

type badRequestErrorFixture struct {
	Status string `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

type badRequestBodyFixture struct {
	Errors []badRequestErrorFixture `json:"errors"`
}

func TestWriteBadRequest_EscapesControlCharsAsValidJSON(t *testing.T) {
	l, _ := test.NewNullLogger()
	w := httptest.NewRecorder()

	detail := "bad\x07value"
	WriteBadRequest(l, w, detail)

	if w.Code != 400 {
		t.Fatalf("status: got %d, want 400", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type: got %q, want application/json", ct)
	}

	var body badRequestBodyFixture
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body did not round-trip through json.Unmarshal: %v (body=%q)", err, w.Body.String())
	}

	if len(body.Errors) != 1 {
		t.Fatalf("errors: got %d entries, want 1", len(body.Errors))
	}
	e := body.Errors[0]
	if e.Status != "400" || e.Title != "Bad Request" || e.Detail != detail {
		t.Fatalf("error object: %+v, want detail=%q", e, detail)
	}
}

func quietLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return l
}

func TestWriteErrorResponseNoClassifierIs500(t *testing.T) {
	RegisterTransientErrorClassifier(nil)
	rec := httptest.NewRecorder()
	WriteErrorResponse(quietLogger())(rec)(errors.New("boom"))
	if rec.Code != 500 {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"500"`) {
		t.Fatalf("expected JSON:API 500 body, got %s", rec.Body.String())
	}
}

func TestWriteErrorResponseTransientIs503(t *testing.T) {
	RegisterTransientErrorClassifier(func(error) bool { return true })
	defer RegisterTransientErrorClassifier(nil)
	rec := httptest.NewRecorder()
	WriteErrorResponse(quietLogger())(rec)(errors.New("pool exhausted"))
	if rec.Code != 503 {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "1" {
		t.Fatalf("expected Retry-After: 1, got %q", got)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"status":"503"`) || !strings.Contains(body, `"title":"temporarily unavailable"`) {
		t.Fatalf("unexpected 503 body: %s", body)
	}
}

func TestWriteErrorResponseNonTransientIs500(t *testing.T) {
	RegisterTransientErrorClassifier(func(error) bool { return false })
	defer RegisterTransientErrorClassifier(nil)
	rec := httptest.NewRecorder()
	WriteErrorResponse(quietLogger())(rec)(errors.New("real bug"))
	if rec.Code != 500 {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "" {
		t.Fatalf("500 must not carry Retry-After, got %q", got)
	}
}
