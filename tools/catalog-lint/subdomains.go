package main

import "regexp"

type subdomainRule struct {
	path    string
	typ     string
	pattern *regexp.Regexp
}

// All subdomain expectations. Mirrors the per-service Subdomain implementations.
var rules = []subdomainRule{
	{path: "drops/monsters", typ: "monster-drop", pattern: regexp.MustCompile(`^monster-(\d+)\.json$`)},
	{path: "drops/continents", typ: "continent-drop", pattern: regexp.MustCompile(`^continent-(-?\d+)\.json$`)},
	{path: "drops/reactors", typ: "reactor-drop", pattern: regexp.MustCompile(`^reactor-(\d+)\.json$`)},
	{path: "gachapons", typ: "gachapon", pattern: regexp.MustCompile(`^gachapon-(.+)\.json$`)},
	{path: "gachapons/_global", typ: "gachapon-pool", pattern: nil},
	{path: "map-actions/onUserEnter", typ: "map-action", pattern: regexp.MustCompile(`^map-(.+)\.json$`)},
	{path: "map-actions/onFirstUserEnter", typ: "map-action", pattern: regexp.MustCompile(`^map-(.+)\.json$`)},
	{path: "portal-actions/portals", typ: "portal-action", pattern: regexp.MustCompile(`^portal-(.+)\.json$`)},
	{path: "reactor-actions/reactors", typ: "reactor-action", pattern: regexp.MustCompile(`^reactor-(.+)\.json$`)},
	{path: "npc-conversations/npc", typ: "npc-conversation", pattern: regexp.MustCompile(`^npc-(\d+)\.json$`)},
	{path: "npc-conversations/quests", typ: "quest-conversation", pattern: regexp.MustCompile(`^quest-(\d+)\.json$`)},
	{path: "npc-conversations/items", typ: "item-conversation", pattern: regexp.MustCompile(`^item-(\d+)\.json$`)},
	{path: "npc-shops/shops", typ: "npc-shop", pattern: regexp.MustCompile(`^shop-(\d+)\.json$`)},
	{path: "party-quests/definitions", typ: "party-quest-definition", pattern: regexp.MustCompile(`^party-quest-(.+)\.json$`)},
	{path: "events/definitions", typ: "event-definition", pattern: regexp.MustCompile(`^event-(.+)\.json$`)},
	// Version-agnostic transport configuration, seeded from
	// deploy/seed/shared/all by atlas-tenants (task-189). pattern is nil
	// because the filename does not encode the entity id — the id lives
	// in data.id (e.g. flight-temple-of-time-leafre.json holds
	// "temple-of-time-return-flight").
	{path: "routes", typ: "routes", pattern: nil},
	{path: "vessels", typ: "vessels", pattern: nil},
	{path: "instance-routes", typ: "instance-routes", pattern: nil},
	// widgets fixture used in tests
	{path: "widgets", typ: "widget", pattern: regexp.MustCompile(`^widget-(\d+)\.json$`)},
}

func ruleFor(relDir string) (subdomainRule, bool) {
	for _, r := range rules {
		if r.path == relDir {
			return r, true
		}
	}
	return subdomainRule{}, false
}
