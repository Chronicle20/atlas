package requests

import (
	"context"
	"github.com/sirupsen/logrus"
	"net/http"
)

//goland:noinspection GoUnusedExportedFunction
func MakePatchRequest[A any](url string, i interface{}, configurators ...Configurator) Request[A] {
	return func(l logrus.FieldLogger, ctx context.Context) (A, error) {
		return createOrUpdate[A](l, ctx)(http.MethodPatch)(url, i, configurators...)
	}
}
