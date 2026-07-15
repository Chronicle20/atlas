package requests

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
)

// drainWarnPages is the page count past which a single drain logs a warning.
const drainWarnPages = 20

type PageMetaPage struct {
	Number int `json:"number"`
	Size   int `json:"size"`
	Last   int `json:"last"`
}

type PageMeta struct {
	Total int          `json:"total"`
	Page  PageMetaPage `json:"page"`
}

// PagedResponse carries one decoded page. Meta == nil means the response
// had no pagination envelope (an unconverted server): the caller must treat
// Data as the complete collection.
type PagedResponse[A any] struct {
	Data []A
	Meta *PageMeta
}

func withPageParams(rawUrl string, page model.Page) (string, error) {
	u, err := url.Parse(rawUrl)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("page[number]", strconv.Itoa(page.Number))
	q.Set("page[size]", strconv.Itoa(page.Size))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// PagedGetRequest issues a GET with page[number]/page[size] appended
// (existing query params preserved) and decodes both the JSON:API data
// array and the pagination envelope from the same body.
func PagedGetRequest[A any](rawUrl string, page model.Page, configurators ...Configurator) Request[PagedResponse[A]] {
	return func(l logrus.FieldLogger, ctx context.Context) (PagedResponse[A], error) {
		u, err := withPageParams(rawUrl, page)
		if err != nil {
			return PagedResponse[A]{}, err
		}
		// Attach the span + tenant header decorators the non-paged GetRequest
		// applies (decorated.go). Prepended so a caller-supplied configurator
		// can still override them. Without these a tenant-scoped server rejects
		// the request with 400 in ParseTenant.
		configurators = append([]Configurator{
			AddHeaderDecorator(SpanHeaderDecorator(ctx)),
			AddHeaderDecorator(TenantHeaderDecorator(ctx)),
		}, configurators...)
		body, err := getBody(l, ctx)(u, configurators...)
		if err != nil {
			return PagedResponse[A]{}, err
		}
		var env struct {
			Meta *PageMeta `json:"meta"`
		}
		if err = json.Unmarshal(body, &env); err != nil {
			return PagedResponse[A]{}, err
		}
		items, err := unmarshalResponse[[]A](body)
		if err != nil {
			return PagedResponse[A]{}, err
		}
		return PagedResponse[A]{Data: items, Meta: env.Meta}, nil
	}
}

// PagedProvider fetches one page and transforms it, returning the paged
// container. If the server sent no envelope, Total falls back to the item
// count and Page to the requested page.
func PagedProvider[A any, M any](l logrus.FieldLogger, ctx context.Context) func(url string, page model.Page, t model.Transformer[A, M], configurators ...Configurator) model.Provider[model.Paged[M]] {
	return func(url string, page model.Page, t model.Transformer[A, M], configurators ...Configurator) model.Provider[model.Paged[M]] {
		return func() (model.Paged[M], error) {
			resp, err := PagedGetRequest[A](url, page, configurators...)(l, ctx)
			if err != nil {
				return model.Paged[M]{}, err
			}
			ms, err := model.SliceMap[A, M](t)(model.FixedProvider(resp.Data))(model.ParallelMap())()
			if err != nil {
				return model.Paged[M]{}, err
			}
			total, pg := len(ms), page
			if resp.Meta != nil {
				total = resp.Meta.Total
				pg = model.Page{Number: resp.Meta.Page.Number, Size: resp.Meta.Page.Size}
			}
			return model.Paged[M]{Items: ms, Total: total, Page: pg}, nil
		}
	}
}

// DrainProvider is the semantic-"all" fetch: it requests page 1 at pageSize
// and iterates page[number] 2..meta.page.last (re-read each response),
// stopping early on an empty page. If a response carries no envelope, that
// single response is treated as the complete collection — this makes
// consumer-first rollout against unconverted servers safe. The (t, filters)
// tail matches SliceProvider so call-site conversion is mechanical.
func DrainProvider[A any, M any](l logrus.FieldLogger, ctx context.Context) func(url string, pageSize int, t model.Transformer[A, M], filters []model.Filter[M], configurators ...Configurator) model.Provider[[]M] {
	return func(url string, pageSize int, t model.Transformer[A, M], filters []model.Filter[M], configurators ...Configurator) model.Provider[[]M] {
		return func() ([]M, error) {
			var out []M
			last := 1
			for number := 1; number <= last; number++ {
				resp, err := PagedGetRequest[A](url, model.Page{Number: number, Size: pageSize}, configurators...)(l, ctx)
				if err != nil {
					return nil, err
				}
				ms, err := model.SliceMap[A, M](t)(model.FixedProvider(resp.Data))(model.ParallelMap())()
				if err != nil {
					return nil, err
				}
				out = append(out, ms...)
				if resp.Meta == nil {
					break
				}
				if len(resp.Data) == 0 {
					break
				}
				last = resp.Meta.Page.Last
				if number == drainWarnPages && last > drainWarnPages {
					l.Warnf("Drain of [%s] exceeds [%d] pages (total [%d]); consider whether this consumer really needs the full collection.", url, drainWarnPages, resp.Meta.Total)
				}
			}
			return model.FilteredProvider(model.FixedProvider(out), filters)()
		}
	}
}
