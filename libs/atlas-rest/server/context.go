package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

type HandlerDependency struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewHandlerDependency(l logrus.FieldLogger, ctx context.Context) HandlerDependency {
	return HandlerDependency{l: l, ctx: ctx}
}

func (h HandlerDependency) Logger() logrus.FieldLogger {
	return h.l
}

func (h HandlerDependency) Context() context.Context {
	return h.ctx
}

type HandlerContext struct {
	si jsonapi.ServerInformation
}

func NewHandlerContext(si jsonapi.ServerInformation) HandlerContext {
	return HandlerContext{si: si}
}

func (h HandlerContext) ServerInformation() jsonapi.ServerInformation {
	return h.si
}

type GetHandler func(d *HandlerDependency, c *HandlerContext) http.HandlerFunc

type InputHandler[M any] func(d *HandlerDependency, c *HandlerContext, model M) http.HandlerFunc

func ParseInput[M any](d *HandlerDependency, c *HandlerContext, next InputHandler[M]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var model M

		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeBadRequestJSONAPI(d.l, w, "The request body could not be read: "+err.Error())
			return
		}
		defer r.Body.Close()

		err = jsonapi.Unmarshal(body, &model)
		if err != nil {
			d.l.WithError(err).Errorln("Deserializing input", err)
			writeBadRequestJSONAPI(d.l, w, "The request body could not be decoded.")
			return
		}
		next(d, c, model)(w, r)
	}
}

// ParseOptionalInput is ParseInput's sibling for handlers whose request body
// is OPTIONAL: an absent body or a `{}` body decodes to the zero value of M
// rather than failing jsonapi.Unmarshal's own "Source JSON is empty" check.
// Any other body -- not valid JSON, or valid JSON without a JSON:API
// envelope (a top-level "data" object naming a "type") -- is still a 400,
// in the same errors-array shape ParseInput uses. ParseInput itself is
// unchanged; body absence remains an error there.
func ParseOptionalInput[M any](d *HandlerDependency, c *HandlerContext, next InputHandler[M]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var model M

		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeBadRequestJSONAPI(d.l, w, "The request body could not be read: "+err.Error())
			return
		}
		defer func() { _ = r.Body.Close() }()

		trimmed := bytes.TrimSpace(body)
		switch {
		case len(trimmed) == 0, string(trimmed) == "{}":
			next(d, c, model)(w, r)
			return
		case !json.Valid(trimmed):
			writeBadRequestJSONAPI(d.l, w, "The request body could not be decoded.")
			return
		case !hasJSONAPIEnvelope(trimmed):
			writeBadRequestJSONAPI(d.l, w, "The request body must be a JSON:API document with a top-level \"data\" object naming a \"type\".")
			return
		}

		err = jsonapi.Unmarshal(trimmed, &model)
		if err != nil {
			d.l.WithError(err).Errorln("Deserializing input", err)
			writeBadRequestJSONAPI(d.l, w, "The request body could not be decoded.")
			return
		}
		next(d, c, model)(w, r)
	}
}

// hasJSONAPIEnvelope reports whether raw is a JSON:API document with a
// top-level "data" object naming a non-empty "type" -- the minimum
// jsonapi.Unmarshal needs to reach setDataIntoTarget instead of failing at
// its own "Source JSON is empty" check. It does not validate the type name
// or attributes; jsonapi.Unmarshal still does that work.
func hasJSONAPIEnvelope(raw []byte) bool {
	var doc struct {
		Data *struct {
			Type string `json:"type"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return false
	}
	return doc.Data != nil && doc.Data.Type != ""
}
