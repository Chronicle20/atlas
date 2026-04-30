package character

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

const nameValidityPath = "characters/name-validity"

type NameValidityResult struct {
	Valid  bool   `json:"valid"`
	Reason string `json:"reason,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type NameValidityClient interface {
	Check(ctx context.Context, name string, worldId byte) (NameValidityResult, error)
}

type NameValidityClientImpl struct {
	l logrus.FieldLogger
}

func NewNameValidityClient(l logrus.FieldLogger) *NameValidityClientImpl {
	return &NameValidityClientImpl{l: l}
}

func getCharacterBaseRequest() string {
	return requests.RootUrl("CHARACTERS")
}

func (c *NameValidityClientImpl) Check(ctx context.Context, name string, worldId byte) (NameValidityResult, error) {
	base := getCharacterBaseRequest()
	u := fmt.Sprintf("%s%s?name=%s&worldId=%d", base, nameValidityPath, url.QueryEscape(name), worldId)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return NameValidityResult{}, err
	}

	// Apply span and tenant headers the same way the atlas-rest GetRequest helper does.
	requests.SpanHeaderDecorator(ctx)(req.Header)
	requests.TenantHeaderDecorator(ctx)(req.Header)

	c.l.Debugf("Issuing [%s] request to [%s].", req.Method, req.URL)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return NameValidityResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return NameValidityResult{}, fmt.Errorf("name-validity HTTP %d: %s", resp.StatusCode, string(body))
	}

	var out NameValidityResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return NameValidityResult{}, err
	}
	return out, nil
}
