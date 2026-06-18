// Package wallet provides a read-only view of a cash-shop wallet for atlas-mts's
// buy pre-check. atlas-mts does NOT own the wallet — the authoritative wallet
// lives in atlas-cashshop and money moves through the saga's AwardCurrency steps.
// This package only READS the buyer's NX Prepaid balance so the buy flow can
// reject an under-funded purchase before emitting a settlement saga (a fast,
// best-effort pre-check; the saga's debit-first AwardCurrency step remains the
// authoritative enforcement of sufficient funds).
//
// The REST read mirrors atlas-channel's cashshop/wallet requester exactly
// (GET {CASHSHOP}/accounts/{accountId}/wallet -> {accountId,credit,points,prepaid}).
package wallet

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Resource is the cash-shop wallet GET path template (accountId-keyed). It matches
// services/atlas-cashshop/.../wallet/resource.go's GET /accounts/{accountId}/wallet.
const Resource = "accounts/%d/wallet"

// RestModel is the cash-shop wallet payload. It is a flat (relationship-free)
// JSON:API resource, so no Unmarshal*Relations stubs are required. CurrencyType
// mapping (from the saga library): 1=credit, 2=points, 3=prepaid.
type RestModel struct {
	Id        uuid.UUID `json:"-"`
	AccountId uint32    `json:"accountId"`
	Credit    uint32    `json:"credit"`
	Points    uint32    `json:"points"`
	Prepaid   uint32    `json:"prepaid"`
}

func (r RestModel) GetName() string { return "wallets" }

func (r RestModel) GetID() string { return r.Id.String() }

func (r *RestModel) SetID(strId string) error {
	id, err := uuid.Parse(strId)
	if err != nil {
		return err
	}
	r.Id = id
	return nil
}

func getBaseRequest() string {
	return requests.RootUrl("CASHSHOP")
}

func requestByAccountId(accountId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, accountId))
}

// Processor reads cash-shop wallet balances over REST.
type Processor interface {
	// PrepaidBalance reads the account's NX Prepaid balance from the cash-shop
	// wallet (the bucket the buy debit draws from, currencyType=3).
	PrepaidBalance(accountId uint32) (uint32, error)
	// Balance reads the account's two MTS wallet buckets — NX Prepaid (prepaid,
	// currencyType=3) and Maple Points (points, currencyType=2) — in a single
	// read. It backs the GET /accounts/{accountId}/mts/wallet read passthrough and
	// the channel-side MTS_OPERATION2 (CITC::OnQueryCashResult) two-bucket wallet
	// announce. Credit (currencyType=1) is not an MTS bucket and is not surfaced.
	Balance(accountId uint32) (prepaid uint32, points uint32, err error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

func (p *ProcessorImpl) PrepaidBalance(accountId uint32) (uint32, error) {
	rm, err := requestByAccountId(accountId)(p.l, p.ctx)
	if err != nil {
		return 0, err
	}
	return rm.Prepaid, nil
}

func (p *ProcessorImpl) Balance(accountId uint32) (uint32, uint32, error) {
	rm, err := requestByAccountId(accountId)(p.l, p.ctx)
	if err != nil {
		return 0, 0, err
	}
	return rm.Prepaid, rm.Points, nil
}
