package server

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

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
