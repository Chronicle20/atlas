package templates

import (
	"testing"
)

// worldTransferAliasGroups mirrors the alias table in
// docs/tasks/task-227-cash-name-change-world-transfer/
// bug-world-transfer-eligibility-reasons.md ("Rulings" ruling 2, "The alias
// table"). Each seeded reason key must resolve to the SAME numeric code as
// its anchor name, in that template's own "errors" table - the numbers are
// not portable across templates, only the names are.
var worldTransferAliasGroups = map[string][]string{
	"CANNOT_TRANSFER_TO_SAME_WORLD":  {"world_same"},
	"CANNOT_TRANSFER_TO_NEW_WORLD":   {"world_full", "world_unknown"},
	"CANNOT_TRANSFER_NO_EMPTY_SLOTS": {"no_character_slot"},
	"PLEASE_TRY_AGAIN":               {"check_unavailable", "unknown_error"},
	"CANNOT_TRANSFER_OUT": {
		"name_taken", "banned", "is_guild_master", "is_gm",
		"in_family", "trade_open", "merchant_open", "mts_listings_open",
	},
}

// findErrorsTable returns the "errors" options map for the writer whose
// errors table contains CANNOT_TRANSFER_TO_SAME_WORLD - the cash shop
// operation writer, identified by content rather than by opCode/index,
// because the writer's position in the socket table differs on every
// version. ok is false when no writer's errors table carries that name (the
// jms_185_1 case) or the template has no "errors" table anywhere at all (the
// gms_12_1 case).
func findErrorsTable(rm RestModel) (map[string]interface{}, bool) {
	for _, w := range rm.Socket.Writers {
		generic, ok := w.Options["errors"]
		if !ok {
			continue
		}
		errs, ok := generic.(map[string]interface{})
		if !ok {
			continue
		}
		if _, ok := errs["CANNOT_TRANSFER_TO_SAME_WORLD"]; ok {
			return errs, true
		}
	}
	return nil, false
}

// TestWorldTransferReasonAliasesResolveToAnchorCode is task-227 step 5's
// seeding test: for every shipped template that carries a
// CANNOT_TRANSFER_TO_SAME_WORLD-bearing errors table, every alias key present
// must resolve to the exact same numeric code as the anchor name it was
// seeded from. It also documents the two exceptions the alias table cannot
// cover: template_gms_12_1.json (no errors table at all) and
// template_jms_185_1.json (an errors table with none of the CANNOT_TRANSFER_*
// names) must NOT carry any alias - there is no honest code to invent for
// them, and worldTransferRejectionReason's fallback in
// atlas-channel/socket/handler/cash_shop_operation.go handles that gap
// explicitly at runtime instead.
func TestWorldTransferReasonAliasesResolveToAnchorCode(t *testing.T) {
	c := LoadCatalog(testLogger(), seedTemplatesDir())

	seededCount := 0
	for _, e := range c.Entries() {
		errs, ok := findErrorsTable(e.Model)
		if !ok {
			// gms_12_1 (no errors table) and jms_185_1 (errors table without
			// the CANNOT_TRANSFER_* names) land here. Nothing to assert.
			continue
		}

		for anchor, aliases := range worldTransferAliasGroups {
			anchorCode, anchorPresent := errs[anchor]
			for _, alias := range aliases {
				aliasCode, aliasPresent := errs[alias]
				if !aliasPresent {
					// gms_48_1/gms_61_1 lack CANNOT_TRANSFER_NO_EMPTY_SLOTS
					// and PLEASE_TRY_AGAIN entirely, and gms_72_1 lacks
					// PLEASE_TRY_AGAIN - the corresponding aliases were
					// correctly not seeded on those templates. Not an error.
					continue
				}
				if !anchorPresent {
					t.Errorf("%s: alias [%s] = %v but anchor [%s] is absent from this template's errors table", e.FileName, alias, aliasCode, anchor)
					continue
				}
				if aliasCode != anchorCode {
					t.Errorf("%s: alias [%s] = %v, want %v (anchor [%s])", e.FileName, alias, aliasCode, anchorCode, anchor)
				}
				seededCount++
			}
		}
	}

	if seededCount == 0 {
		t.Fatal("no world-transfer reason aliases were found on any shipped template - seeding regressed")
	}
}

// TestWorldTransferUnmappableTemplatesCarryNoAliases pins the two documented
// exceptions by name, so a future re-derivation of the alias table cannot
// silently invent codes for them.
func TestWorldTransferUnmappableTemplatesCarryNoAliases(t *testing.T) {
	c := LoadCatalog(testLogger(), seedTemplatesDir())

	for _, name := range []string{"template_gms_12_1.json", "template_jms_185_1.json"} {
		var entry CatalogEntry
		found := false
		for _, cand := range c.Entries() {
			if cand.FileName == name {
				entry = cand
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("shipped catalog has no entry for %s", name)
		}
		if _, found := findErrorsTable(entry.Model); found {
			t.Errorf("%s: expected no CANNOT_TRANSFER_TO_SAME_WORLD-bearing errors table, found one - update the aliasing fallback in cash_shop_operation.go if this template gained transfer codes", name)
		}
	}
}
