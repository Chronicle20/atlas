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
	"errors"
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

// createRequest POSTs a new cash-shop wallet for the account (JSON:API enveloped
// by the requests layer). Matches cashshop's POST /accounts/{accountId}/wallet
// (handleCreateWallet), which reads accountId from the path and credit/points/
// prepaid from the body.
func createRequest(accountId uint32, rm RestModel) requests.Request[RestModel] {
	return requests.PostRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, accountId), rm)
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
	// EnsureWallet guarantees the account has a cash-shop wallet, creating one
	// (with the given starting balances) only if none exists. It exists for the
	// test-seed flow: a seeded listing's synthetic seller must have a wallet or the
	// buy's seller-points credit fails. Idempotent — a wallet that already exists is
	// left untouched (cashshop's create is a plain INSERT and would 500 on a
	// duplicate). Real accounts get their wallet from the account-created event.
	EnsureWallet(accountId uint32, credit uint32, points uint32, prepaid uint32) error
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

func (p *ProcessorImpl) EnsureWallet(accountId uint32, credit uint32, points uint32, prepaid uint32) error {
	// A successful GET means the wallet already exists — leave it as-is.
	if _, err := requestByAccountId(accountId)(p.l, p.ctx); err == nil {
		return nil
	} else if !errors.Is(err, requests.ErrNotFound) {
		return err
	}
	rm := RestModel{AccountId: accountId, Credit: credit, Points: points, Prepaid: prepaid}
	_, err := createRequest(accountId, rm)(p.l, p.ctx)
	return err
}
