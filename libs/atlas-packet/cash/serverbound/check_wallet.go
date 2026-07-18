package serverbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

const CashShopCheckWalletHandle = "CashShopCheckWalletHandle"

// CheckWallet - CCashShop::SendCheckWallet
// packet-audit:fname CCashShop::TrySendQueryCashRequest
type CheckWallet struct{}

func (m CheckWallet) Operation() string {
	return CashShopCheckWalletHandle
}

func (m CheckWallet) String() string {
	return ""
}

func (m CheckWallet) Encode(_ logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	return func(options map[string]interface{}) []byte {
		return []byte{}
	}
}

func (m *CheckWallet) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
