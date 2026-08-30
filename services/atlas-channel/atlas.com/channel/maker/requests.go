// Package maker is the atlas-channel client for atlas-maker's POST
// /characters/{id}/maker/crafts (Task 24). The channel forwards the
// decoded MAKER_SKILL request verbatim -- atlas-maker owns every validation
// rule (PRD §5); this package is transport only.
package maker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jtumidanski/api2go/jsonapi"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const craftsResource = "characters/%d/maker/crafts"

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MAKER")
}

// CraftRequest mirrors atlas-maker's craft.CraftRequestRestModel (the POST
// /crafts body). Field names and JSON tags are the wire contract with
// atlas-maker's craft package -- keep them in lockstep.
type CraftRequest struct {
	Id             string   `json:"-"`
	Mode           uint32   `json:"mode"`
	WorldId        byte     `json:"worldId"`
	ChannelId      byte     `json:"channelId"`
	TargetItemId   uint32   `json:"targetItemId,omitempty"`
	UseCatalyst    bool     `json:"useCatalyst,omitempty"`
	GemItemIds     []uint32 `json:"gemItemIds,omitempty"`
	LeftoverItemId uint32   `json:"leftoverItemId,omitempty"`
	EquipItemId    uint32   `json:"equipItemId,omitempty"`
	SlotPos        int16    `json:"slotPos,omitempty"`
}

func (r CraftRequest) GetName() string           { return "makerCrafts" }
func (r CraftRequest) GetID() string             { return r.Id }
func (r *CraftRequest) SetID(idStr string) error { r.Id = idStr; return nil }

// CraftResponse mirrors atlas-maker's CraftResponseRestModel: the accepted
// craft's saga transaction id.
type CraftResponse struct {
	Id            string `json:"-"`
	TransactionId string `json:"transactionId"`
}

func (r CraftResponse) GetName() string           { return "makerCrafts" }
func (r CraftResponse) GetID() string             { return r.Id }
func (r *CraftResponse) SetID(idStr string) error { r.Id = idStr; return nil }

// craftError carries the raw PRD §5 `code` from a rejected POST /crafts
// response, so Processor.Create's caller can log which rule fired without
// the channel importing atlas-maker's internal craft package across the
// service boundary.
type craftError struct {
	code string
}

func (e craftError) Error() string { return e.code }

// craftErrorDocument is the minimal shape of atlas-maker's JSON:API error
// body (services/atlas-maker/atlas.com/maker/craft/errors.go's
// errorDocument) -- only the stable `code` is read.
type craftErrorDocument struct {
	Errors []struct {
		Code string `json:"code"`
	} `json:"errors"`
}

func decodeErrorCode(body []byte) string {
	var doc craftErrorDocument
	if err := json.Unmarshal(body, &doc); err != nil || len(doc.Errors) == 0 {
		return ""
	}
	return doc.Errors[0].Code
}

// requestCreateCraft POSTs the craft request verbatim to atlas-maker and
// returns its accepted response or a craftError carrying the rejected
// request's stable code. The generic atlas-rest/requests wrapper collapses
// every non-2xx/404/409 status to "unknown error" (it never exposes the
// response body), which would destroy the PRD §5 code atlas-maker sends for
// every 422 -- so this call is made directly.
func requestCreateCraft(ctx context.Context, characterId uint32, input CraftRequest) (CraftResponse, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return CraftResponse{}, err
	}
	url := fmt.Sprintf(root+craftsResource, characterId)

	reqBody, err := jsonapi.Marshal(input)
	if err != nil {
		return CraftResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return CraftResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/vnd.api+json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return CraftResponse{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return CraftResponse{}, err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted:
		var out CraftResponse
		if len(respBody) == 0 {
			return out, nil
		}
		if err := jsonapi.Unmarshal(respBody, &out); err != nil {
			return CraftResponse{}, err
		}
		return out, nil
	default:
		code := decodeErrorCode(respBody)
		if code == "" {
			code = CodeUnknown
		}
		return CraftResponse{}, craftError{code: code}
	}
}
