package pendingchange

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// httpClient mirrors the default client used elsewhere in this package's
// dependency (libs/atlas-rest/requests) for consistent timeout behavior.
var httpClient = &http.Client{Timeout: 15 * time.Second}

// errorObject / errorDocument decode the JSON:API error shape written by
// atlas-character's pending_change resource (writeReasonError in
// services/atlas-character/atlas.com/character/pending_change/resource.go).
type errorObject struct {
	Status string `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}

type errorDocument struct {
	Errors []errorObject `json:"errors"`
}

// StatusError carries the HTTP status code and, when present, the JSON:API
// error document's detail string from a non-2xx response. Unlike
// requests.MakePostRequest's backing createOrUpdate, it preserves the body
// on a 422 so the caller can read the rejection reason (libs/atlas-rest's
// createOrUpdate discards the body and returns a bare "unknown error" for
// any status outside 200/201/202/204/400/404/409).
type StatusError struct {
	StatusCode int
	Detail     string
}

func (e *StatusError) Error() string {
	if e.Detail != "" {
		return e.Detail
	}
	return http.StatusText(e.StatusCode)
}

// postCreate issues the pending-change creation POST directly via net/http
// (rather than requests.MakePostRequest) so a 422 response body can be read
// for its `detail` reason. Header parity with the shared helper is
// preserved via requests.TenantHeaderDecorator and requests.SpanHeaderDecorator.
func postCreate(l logrus.FieldLogger, ctx context.Context) func(url string, input CreateInputRestModel) (RestModel, error) {
	return func(url string, input CreateInputRestModel) (RestModel, error) {
		var result RestModel

		jsonReq, err := jsonapi.Marshal(input)
		if err != nil {
			return result, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonReq))
		if err != nil {
			l.WithError(err).Errorf("Error creating request.")
			return result, err
		}

		requests.TenantHeaderDecorator(ctx)(req.Header)
		requests.SpanHeaderDecorator(ctx)(req.Header)

		l.Debugf("Issuing [%s] request to [%s].", http.MethodPost, req.URL)
		resp, err := httpClient.Do(req)
		if err != nil {
			l.WithError(err).Warnf("Failed calling [%s] on [%s].", http.MethodPost, url)
			return result, err
		}
		defer func() { _ = resp.Body.Close() }()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			l.WithError(err).Warnf("Failed reading response from [%s] on [%s].", http.MethodPost, url)
			return result, err
		}

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusAccepted {
			if len(body) == 0 {
				return result, nil
			}
			if err := jsonapi.Unmarshal(body, &result); err != nil {
				return result, err
			}
			return result, nil
		}

		detail := ""
		var doc errorDocument
		if len(body) > 0 && json.Unmarshal(body, &doc) == nil && len(doc.Errors) > 0 {
			detail = doc.Errors[0].Detail
		}
		return result, &StatusError{StatusCode: resp.StatusCode, Detail: detail}
	}
}

// postCancel issues the self-scoped cancel POST, mirroring postCreate's
// direct net/http use (rather than requests.MakePostRequest) so a non-2xx
// response body can be read for its `detail` reason -- including the 404
// "nothing pending" case, which is not an infrastructure failure.
func postCancel(l logrus.FieldLogger, ctx context.Context) func(url string, input CancelInputRestModel) (RestModel, error) {
	return func(url string, input CancelInputRestModel) (RestModel, error) {
		var result RestModel

		jsonReq, err := jsonapi.Marshal(input)
		if err != nil {
			return result, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonReq))
		if err != nil {
			l.WithError(err).Errorf("Error creating request.")
			return result, err
		}

		requests.TenantHeaderDecorator(ctx)(req.Header)
		requests.SpanHeaderDecorator(ctx)(req.Header)

		l.Debugf("Issuing [%s] request to [%s].", http.MethodPost, req.URL)
		resp, err := httpClient.Do(req)
		if err != nil {
			l.WithError(err).Warnf("Failed calling [%s] on [%s].", http.MethodPost, url)
			return result, err
		}
		defer func() { _ = resp.Body.Close() }()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			l.WithError(err).Warnf("Failed reading response from [%s] on [%s].", http.MethodPost, url)
			return result, err
		}

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusAccepted {
			if len(body) == 0 {
				return result, nil
			}
			if err := jsonapi.Unmarshal(body, &result); err != nil {
				return result, err
			}
			return result, nil
		}

		detail := ""
		var doc errorDocument
		if len(body) > 0 && json.Unmarshal(body, &doc) == nil && len(doc.Errors) > 0 {
			detail = doc.Errors[0].Detail
		}
		return result, &StatusError{StatusCode: resp.StatusCode, Detail: detail}
	}
}

// asStatusError unwraps a *StatusError from err, if any.
func asStatusError(err error) (*StatusError, bool) {
	var se *StatusError
	if errors.As(err, &se) {
		return se, true
	}
	return nil, false
}
