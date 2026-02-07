package ban

import (
	"context"

	"github.com/sirupsen/logrus"
)

func CheckBan(l logrus.FieldLogger, ctx context.Context, ip string, hwid string, accountId uint32) (CheckRestModel, error) {
	return requestCheckBan(ip, hwid, accountId)(l, ctx)
}
