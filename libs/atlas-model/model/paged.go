package model

// Page identifies one page of a collection. Number is 1-based.
type Page struct {
	Number int
	Size   int
}

// Paged carries one page of items together with the pre-paging total of
// rows matching the scope and the page that produced Items.
type Paged[T any] struct {
	Items []T
	Total int
	Page  Page
}

// MapPaged lifts an item transform over the Paged container, preserving
// Total/Page. Composes exactly like SliceMap:
//
//	MapPaged(f)(provider)(ParallelMap())
//
// Decoration needs no separate primitive:
//
//	MapPaged(Decorate[M](decorators))(p)(ParallelMap())
func MapPaged[E any, M any](transformer Transformer[E, M]) func(provider Provider[Paged[E]]) func(configurators ...MapFuncConfigurator) Provider[Paged[M]] {
	return func(provider Provider[Paged[E]]) func(configurators ...MapFuncConfigurator) Provider[Paged[M]] {
		return func(configurators ...MapFuncConfigurator) Provider[Paged[M]] {
			return func() (Paged[M], error) {
				pe, err := provider()
				if err != nil {
					return Paged[M]{}, err
				}
				items, err := SliceMap[E, M](transformer)(FixedProvider(pe.Items))(configurators...)()
				if err != nil {
					return Paged[M]{}, err
				}
				return Paged[M]{Items: items, Total: pe.Total, Page: pe.Page}, nil
			}
		}
	}
}
