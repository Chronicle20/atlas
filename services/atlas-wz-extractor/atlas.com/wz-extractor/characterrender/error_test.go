package characterrender

import (
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestWriteErrorJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, 400, ErrorBody{
		Code:   "unknown-template-id",
		Title:  "Equipment templateId not present",
		Detail: "templateId 1002357 not found",
		Meta:   map[string]any{"templateId": 1002357},
	})
	if rec.Code != 400 {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/vnd.api+json" {
		t.Fatalf("content-type = %q", ct)
	}
	var got struct {
		Errors []struct {
			Status string         `json:"status"`
			Code   string         `json:"code"`
			Title  string         `json:"title"`
			Detail string         `json:"detail"`
			Meta   map[string]any `json:"meta"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Errors) != 1 {
		t.Fatalf("errors len = %d", len(got.Errors))
	}
	if got.Errors[0].Code != "unknown-template-id" {
		t.Fatalf("code = %q", got.Errors[0].Code)
	}
	if got.Errors[0].Status != strconv.Itoa(400) {
		t.Fatalf("status = %q", got.Errors[0].Status)
	}
}
