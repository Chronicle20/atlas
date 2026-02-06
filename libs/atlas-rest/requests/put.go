package requests

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"
)

//goland:noinspection GoUnusedExportedFunction
func MakePutRequest[A any](url string, i interface{}, configurators ...Configurator) Request[A] {
	return func(l logrus.FieldLogger, ctx context.Context) (A, error) {
		return createOrUpdate[A](l, ctx)(http.MethodPut)(url, i, configurators...)
	}
}
