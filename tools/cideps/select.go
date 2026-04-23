package main

type SelectInput struct {
	ChangedLibs     []string
	ChangedServices []string
	ForceAll        bool
}

type Selection struct {
	Services []string
	Libs     []string
}

// Select computes the affected-module set.
//
// Rules:
//   - ForceAll → every service and library in the graph.
//   - Otherwise a service is affected when it is in ChangedServices or when
//     its lib-closure intersects ChangedLibs.
//   - Otherwise a library is affected when it is in ChangedLibs or when its
//     lib-closure intersects ChangedLibs.
//   - Non-Go services (atlas-ui, atlas-assets) aren't in the graph but are still
//     returned when present in ChangedServices; the enrichment step filters by
//     services.json type/docker_image.
//   - Unknown names in ChangedLibs are ignored silently (graph filter).
func Select(g *Graph, in SelectInput) Selection {
	if in.ForceAll {
		return Selection{Services: g.Services(), Libs: g.Libs()}
	}

	changedLibs := make(map[string]struct{})
	for _, n := range in.ChangedLibs {
		if _, ok := g.deps[n]; ok {
			changedLibs[n] = struct{}{}
		}
	}

	affectedSvcs := make(map[string]struct{})
	for _, n := range in.ChangedServices {
		affectedSvcs[n] = struct{}{}
	}
	for _, svc := range g.Services() {
		if _, done := affectedSvcs[svc]; done {
			continue
		}
		for _, lib := range g.Closure(svc) {
			if _, hit := changedLibs[lib]; hit {
				affectedSvcs[svc] = struct{}{}
				break
			}
		}
	}

	affectedLibs := make(map[string]struct{})
	for lib := range changedLibs {
		affectedLibs[lib] = struct{}{}
	}
	for _, lib := range g.Libs() {
		if _, done := affectedLibs[lib]; done {
			continue
		}
		for _, dep := range g.Closure(lib) {
			if _, hit := changedLibs[dep]; hit {
				affectedLibs[lib] = struct{}{}
				break
			}
		}
	}

	return Selection{
		Services: sortedKeys(affectedSvcs),
		Libs:     sortedKeys(affectedLibs),
	}
}
