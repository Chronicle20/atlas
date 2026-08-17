package character

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const nameValidityPath = "characters/name-validity"

// NameScope selects the uniqueness scope atlas-character applies to a name
// check. Character creation uses WORLD; the cash-shop rename uses TENANT
// (task-227 FR-3.2), which is deliberately stricter so a renamed character can
// never collide with a name held in another world of the same tenant.
type NameScope string

const (
	NameScopeWorld  NameScope = "WORLD"
	NameScopeTenant NameScope = "TENANT"
)

// Reason values atlas-character returns on an invalid name
// (services/atlas-character/.../character/processor.go CheckNameValidity). They
// are mirrored here as constants so the handler's mapping onto the packet
// taxonomy is a table of named values rather than bare strings.
const (
	NameReasonLength    = "length"
	NameReasonRegex     = "regex"
	NameReasonDuplicate = "duplicate"
	NameReasonReserved  = "reserved"
)

// NameValidityResult is atlas-character's GET /characters/name-validity
// response. Valid is the only field set on success; Reason carries one of the
// NameReason* values otherwise.
type NameValidityResult struct {
	Valid    bool   `json:"valid"`
	Reason   string `json:"reason,omitempty"`
	Detail   string `json:"detail,omitempty"`
	Reserved bool   `json:"reserved,omitempty"`
}

// checkNameValidity issues the name-validity request.
//
// atlas-character's GET /characters/name-validity returns PLAIN JSON, not
// JSON:API, so this cannot go through requests.GetRequest[T] the way the rest
// of this package does — that helper runs jsonapi.Unmarshal over the body and
// would fail. The decorators applied below are the same ones GetRequest
// applies, kept in step with
// services/atlas-character-factory/.../character/name_validity_requests.go,
// which makes the identical call for character creation.
func checkNameValidity(l logrus.FieldLogger, ctx context.Context, name string, worldId world.Id, scope NameScope) (NameValidityResult, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return NameValidityResult{}, err
	}
	u := fmt.Sprintf("%s%s?name=%s&worldId=%d&scope=%s", root, nameValidityPath, url.QueryEscape(name), worldId, scope)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return NameValidityResult{}, err
	}
	requests.SpanHeaderDecorator(ctx)(req.Header)
	requests.TenantHeaderDecorator(ctx)(req.Header)

	l.Debugf("Issuing [%s] request to [%s].", req.Method, req.URL)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return NameValidityResult{}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return NameValidityResult{}, fmt.Errorf("name-validity HTTP %d: %s", resp.StatusCode, string(body))
	}

	var out NameValidityResult
	if err = json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return NameValidityResult{}, err
	}
	return out, nil
}
