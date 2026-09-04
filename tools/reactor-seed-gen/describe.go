package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// minDescriptionLen is the shortest cleaned comment describe() will accept
// without consulting the override table.
const minDescriptionLen = 6

var (
	licenseParagraphPattern = regexp.MustCompile(`(?s)This file is part of.*?<http://www\.gnu\.org/licenses/>\.`)
	authorTagPattern        = regexp.MustCompile(`@[Aa]uthor\s+\S+`)
	purposeTagPattern       = regexp.MustCompile(`@purpose\s+`)
	mapTagPattern           = regexp.MustCompile(`@map\s+(?:[A-Z][A-Za-z]*\s*)+`)
	remainingTagPattern     = regexp.MustCompile(`@\w+\s*`)
	urlPattern              = regexp.MustCompile(`\bhttps?://\S+`)
	gnuLicenseUrlPattern    = regexp.MustCompile(`www\.gnu\.org/licenses/>\.`)
	blogspotPattern         = regexp.MustCompile(`mymapleland\.blogspot\.com/\S+`)
	referenceLeadInPattern  = regexp.MustCompile(`\b\w+ reference:\s*`)
	jsPrefixPattern         = regexp.MustCompile(`^[\s*]*\d+\.js[:\-]?\s*`)
	jsReferencePattern      = regexp.MustCompile(`\b\d+\.js\b`)
	whitespacePattern       = regexp.MustCompile(`\s+`)
)

// describe derives a seed description from a reactor's raw source comment.
// The cleaner strips AGPL boilerplate, author and purpose tags, URL debris,
// the "<id>.js:" prefix and "*" decorations. When the result is empty or
// shorter than minDescriptionLen, the override table must supply one;
// there is no fallback that invents text.
func describe(reactorId, comment string) (string, error) {
	cleaned := clean(comment)
	if len(cleaned) < minDescriptionLen {
		if override, ok := descriptionOverrides[reactorId]; ok {
			return override, nil
		}
		return "", fmt.Errorf("reactor %s: no usable description (cleaned comment %q) and no override in descriptionOverrides", reactorId, cleaned)
	}
	return cleaned, nil
}

// clean applies the boilerplate-stripping pipeline to a raw source comment.
func clean(comment string) string {
	s := comment

	// 1. remove the HeavenMS/OdinMS license paragraph.
	s = licenseParagraphPattern.ReplaceAllString(s, "")

	// 2. remove @-tags. @author's trailing name and @map's trailing map
	// name are metadata about the comment, not the reactor's behaviour, so
	// both the tag and its value are dropped. @purpose's trailing text is
	// the actual description content, so only the tag itself is stripped
	// and the text after it is kept. remainingTagPattern is a catch-all
	// for any other "@tag" this corpus does not otherwise name — it must
	// run last, after the tag-specific patterns, so it never eats a value
	// one of them needs to preserve.
	s = authorTagPattern.ReplaceAllString(s, "")
	s = mapTagPattern.ReplaceAllString(s, "")
	s = purposeTagPattern.ReplaceAllString(s, "")
	s = remainingTagPattern.ReplaceAllString(s, "")

	// 3. remove URL debris. urlPattern runs before blogspotPattern so a
	// scheme-prefixed link ("http://mymapleland.blogspot.com/...") is
	// removed whole, including its scheme, rather than leaving a dangling
	// "http://" behind for the schemeless blogspotPattern to skip over.
	// referenceLeadInPattern then drops an orphaned "<word> reference:"
	// label whose target URL has already been stripped.
	s = gnuLicenseUrlPattern.ReplaceAllString(s, "")
	s = urlPattern.ReplaceAllString(s, "")
	s = blogspotPattern.ReplaceAllString(s, "")
	s = referenceLeadInPattern.ReplaceAllString(s, "")

	// 4. remove a leading "<digits>.js:" or "<digits>.js -" prefix (allowing
	// for "*" decorations before it), then any remaining "<digits>.js"
	// filename reference anywhere, including the reactor's own id.
	s = jsPrefixPattern.ReplaceAllString(s, "")
	s = jsReferencePattern.ReplaceAllString(s, "")

	// 5. strip "*" decorations and collapse whitespace.
	s = strings.ReplaceAll(s, "*", " ")
	s = whitespacePattern.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)

	// 6. upper-case the first rune.
	if s != "" {
		r := []rune(s)
		r[0] = unicode.ToUpper(r[0])
		s = string(r)
	}

	return s
}

// descriptionOverrides supplies a description for every reactor whose source
// comment cleans to nothing or to something uninformative. An entry is
// REQUIRED whenever the cleaner falls short — describe() returns an error
// rather than emit a guess, so this table is the one place human judgment
// enters the generated corpus. Review this, not the 1,749 files.
var descriptionOverrides = map[string]string{
	// Identical act() { rm.dropItems(...) }, no source comment at all. No
	// location name appears anywhere in this corpus for these ids.
	"1002008": "Box - drops items",
	"1002009": "Box - drops items",

	// Relic room family (1021000-1021002 "relic room fail", 1022000 "relic
	// complete", 1022002 "Construction Site" fire hydrant). 1022001 shares
	// 1022000's act() { rm.dropItems() } and sits directly between the two
	// labeled relic-room entries. No location name for this family appears
	// anywhere in this corpus.
	"1022001": "Relic room - drops reward items",

	// No source comment and no location context in this corpus; act() is a
	// bare drop.
	"1032000": "Box - drops items",
	"1052000": "Box - drops items",
	"1052001": "Box - sprays items",
	"1052002": "Box - sprays items",

	// Nautilus bottom shells family: 1202002's real comment reads "Nautilus
	// bottom shells"; 1202000, 1202003, 1202004 share the same act() {
	// rm.dropItems() } and id prefix. 1202000's own comment ("@Author
	// dangoron * * 1102000.js") cleans to the wrong id's filename, not a
	// description, so it needs the override too.
	"1202000": "Nautilus bottom shells - drops items",
	"1202003": "Nautilus bottom shells - drops items",
	"1202004": "Nautilus bottom shells - drops items",

	"1209001": "Box - drops items",
	"1402000": "Box - drops items",

	// Bare-id heading "200" and the "200000"-"200009" family: identical
	// act() { rm.dropItems() }, no usable comment.
	"200":     "Box - drops items",
	"200000":  "Box - drops items",
	"200001":  "Box - drops items",
	"200002":  "Box - drops items",
	"200003":  "Box - drops items",
	"200004":  "Box - drops items",
	"200005":  "Box - drops items",
	"200006":  "Box - drops items",
	"200007":  "Box - drops items",
	"200008":  "Box - drops items",
	"200009":  "Box - drops items",
	"2052001": "Box - drops items",
	"2092001": "Box - drops items",

	// Zakum Party Quest chest/rock family (2112000-2112017 are almost all
	// labeled "Zakum Party Quest Chest/Rock - drops ..."); 2112015 shares
	// their bare act() { rm.dropItems() } and sits inside that id range.
	"2112015": "Zakum Party Quest - drops an item",

	// No source comment; rm.weakenAreaBoss(6090001, "...Snow Witch...").
	// No location name for this family appears anywhere in this corpus (the
	// neighboring 2119000-2119003 Tombstones are a different location,
	// "Forest of Dead Trees").
	"2119004": "Altar - weakens the Snow Witch",
	"2119005": "Altar - weakens the Snow Witch",
	"2119006": "Altar - weakens the Snow Witch",

	// rm.weakenAreaBoss(6090003, "...Scholar Ghost...").
	"2229009": "Altar - weakens the Scholar Ghost",

	// No source comment; rm.weakenAreaBoss(6090004, "Rurumo has been
	// poisoned..."). No location name for this family appears anywhere in
	// this corpus; the 6102002-6102005 CWKPQ family runs a different
	// operation (sprayItems, not weakenAreaBoss) and is not adjacent.
	"2619003": "Altar - weakens Rurumo",
	"2619004": "Altar - weakens Rurumo",
	"2619005": "Altar - weakens Rurumo",

	"3102000": "Box - drops items",

	// No source comment; bare act() { rm.dropItems() }. The adjacent
	// 6102002-6102005 family is labeled "Drops CWKPQ chest bonuses", but
	// those ids run rm.sprayItems(...), a different operation, so that
	// label does not ground a name for this id. No location/quest name
	// appears anywhere in this corpus for 6102001 itself.
	"6102001": "Box - drops items",

	// No source comment, no location context in this corpus.
	"6741001": "Spawns a monster",
	"6741015": "Box - drops items",
	"6742014": "Box - sprays items",
	"6802000": "Box - sprays items",
	"6802001": "Box - sprays items",
	"8001000": "Spawns a monster",
}
