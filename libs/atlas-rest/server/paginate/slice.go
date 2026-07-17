package paginate

import "github.com/Chronicle20/atlas/libs/atlas-model/model"

// Slice pages an already-materialized collection (runtime registries,
// document sub-lists). items MUST already be deterministically ordered by
// the caller. Past-end pages return empty Items with the correct Total —
// the Envelope's recovery links handle the UX.
func Slice[T any](items []T, page model.Page) model.Paged[T] {
	total := len(items)
	start := (page.Number - 1) * page.Size
	if start < 0 || start >= total {
		return model.Paged[T]{Items: []T{}, Total: total, Page: page}
	}
	end := start + page.Size
	if end > total {
		end = total
	}
	return model.Paged[T]{Items: items[start:end], Total: total, Page: page}
}
