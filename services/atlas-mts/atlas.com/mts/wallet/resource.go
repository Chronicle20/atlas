package wallet

import (
	"atlas-mts/rest"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
)

// BalanceReader is the read surface the wallet route depends on. The production
// implementation is this package's Processor (a REST read of atlas-cashshop's
// wallet); the resource test injects a stub so the route can be exercised without
// a live cash-shop wallet. The default factory builds the real REST-backed
// Processor.
type BalanceReader interface {
	Balance(accountId uint32) (prepaid uint32, points uint32, err error)
}

// ReaderFactory builds the BalanceReader for a request from the request-scoped
// logger and context (so the outbound REST read carries the tenant/trace
// context). Tests override it to return a stub.
type ReaderFactory func(d *rest.HandlerDependency) BalanceReader

// defaultReaderFactory builds the real REST-backed cash-shop wallet reader.
func defaultReaderFactory(d *rest.HandlerDependency) BalanceReader {
	return NewProcessor(d.Logger(), d.Context())
}

// WalletRestModel is the JSON:API "wallets" read resource: the account's two MTS
// wallet buckets — NX Prepaid and Maple Points. It mirrors the channel-side
// MTS_OPERATION2 (CITC::OnQueryCashResult) two-bucket shape. The account's bare
// credit bucket (currencyType=1) is not an MTS bucket and is intentionally absent.
type WalletRestModel struct {
	Id      string `json:"-"`
	Prepaid uint32 `json:"prepaid"`
	Points  uint32 `json:"points"`
}

func (r WalletRestModel) GetName() string { return "wallets" }
func (r WalletRestModel) GetID() string   { return r.Id }

func (r *WalletRestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

// InitResource registers the wallet read route:
//   - GET /accounts/{accountId}/mts/wallet — the account's two MTS wallet buckets
//
// The route is keyed by accountId (not characterId): the authoritative wallet
// lives in atlas-cashshop and is account-scoped, and atlas-mts has no
// characterId->accountId resolver (the buy flow receives the buyer's accountId
// from its caller). Surfacing the read account-keyed mirrors the cash-shop
// wallet's own GET /accounts/{accountId}/wallet contract and the buy flow's
// account-keyed PrepaidBalance read — no fabricated cross-service resolver.
func InitResource(si jsonapi.ServerInformation) func(db *gorm.DB) server.RouteInitializer {
	return initResource(si, defaultReaderFactory)
}

func initResource(si jsonapi.ServerInformation, rf ReaderFactory) func(db *gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			registerGet := rest.RegisterHandler(l)(db)(si)
			r := router.PathPrefix("/accounts/{accountId}/mts/wallet").Subrouter()
			r.HandleFunc("", registerGet("get_account_wallet", handleGetWallet(rf))).Methods(http.MethodGet)
		}
	}
}

func handleGetWallet(rf ReaderFactory) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseAccountId(d.Logger(), func(accountId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				prepaid, points, err := rf(d).Balance(accountId)
				if err != nil {
					d.Logger().WithError(err).Errorf("Retrieving MTS wallet for account [%d].", accountId)
					server.WriteErrorResponse(d.Logger())(w)(err)
					return
				}

				res := WalletRestModel{
					Id:      strconv.FormatUint(uint64(accountId), 10),
					Prepaid: prepaid,
					Points:  points,
				}

				query := r.URL.Query()
				queryParams := jsonapi.ParseQueryFields(&query)
				server.MarshalResponse[WalletRestModel](d.Logger())(w)(c.ServerInformation())(queryParams)(res)
			}
		})
	}
}
